// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package action_test

import (
	"testing"
	"time"

	"pgregory.net/rapid"

	"go.thesmos.sh/testkit/clock"
	"go.thesmos.sh/testkit/model/action"
	"go.thesmos.sh/testkit/model/timeaware"
)

func TestAdvanceClock(t *testing.T) {
	t.Parallel()

	t.Run("advances both clocks by same amount", func(t *testing.T) {
		t.Parallel()
		origin := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
		sutClk := clock.NewTestClock(origin)
		refClk := clock.NewTestClock(origin)

		a := action.AdvanceClock[string](
			"AdvanceClock",
			func() (*clock.TestClock, *clock.TestClock) { return sutClk, refClk },
			time.Hour,
		)

		rapid.Check(t, func(rt *rapid.T) {
			a.Run(rt, "sut", "ref")
		})

		if sutClk.Now() != refClk.Now() {
			t.Fatalf("clocks diverged: sut=%v ref=%v", sutClk.Now(), refClk.Now())
		}
		if !sutClk.Now().After(origin) {
			t.Fatal("clock should have advanced past origin")
		}
	})

	t.Run("multiple advances accumulate", func(t *testing.T) {
		t.Parallel()
		origin := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
		sutClk := clock.NewTestClock(origin)
		refClk := clock.NewTestClock(origin)

		a := action.AdvanceClock[string](
			"AdvanceClock",
			func() (*clock.TestClock, *clock.TestClock) { return sutClk, refClk },
			time.Minute,
		)

		rapid.Check(t, func(rt *rapid.T) {
			for range 10 {
				a.Run(rt, "sut", "ref")
			}
		})

		if sutClk.Now() != refClk.Now() {
			t.Fatalf("clocks diverged: sut=%v ref=%v", sutClk.Now(), refClk.Now())
		}
	})

	t.Run("nil clocks are safe", func(t *testing.T) {
		t.Parallel()
		a := action.AdvanceClock[string](
			"AdvanceClock",
			func() (*clock.TestClock, *clock.TestClock) { return nil, nil },
			time.Hour,
		)
		rapid.Check(t, func(rt *rapid.T) {
			a.Run(rt, "sut", "ref") // must not panic
		})
	})
}

func TestAdvanceClockSynced(t *testing.T) {
	t.Parallel()

	t.Run("nil barrier degrades to plain advance", func(t *testing.T) {
		t.Parallel()
		sutClk := clock.NewTestClock(time.Unix(0, 0))
		refClk := clock.NewTestClock(time.Unix(0, 0))
		clocks := func() (*clock.TestClock, *clock.TestClock) { return sutClk, refClk }
		a := action.AdvanceClockSynced[string]("AdvanceClock", clocks, time.Second, nil)
		rapid.Check(t, func(rt *rapid.T) {
			a.Run(rt, "sut", "ref")
		})
	})

	t.Run("barrier blocks advance while ops are in-flight", func(t *testing.T) {
		t.Parallel()
		sutClk := clock.NewTestClock(time.Unix(0, 0))
		refClk := clock.NewTestClock(time.Unix(0, 0))
		clocks := func() (*clock.TestClock, *clock.TestClock) { return sutClk, refClk }
		barrier := timeaware.NewBarrier()
		a := action.AdvanceClockSynced[string]("AdvanceClock", clocks, time.Hour, barrier)

		// Hold a fake op for ~5ms; advance must block until release.
		release := barrier.Op()
		baseSUT := sutClk.Now()
		advanceDone := make(chan struct{})
		go func() {
			rapid.Check(t, func(rt *rapid.T) {
				a.Run(rt, "sut", "ref")
			})
			close(advanceDone)
		}()
		time.Sleep(5 * time.Millisecond)
		select {
		case <-advanceDone:
			t.Fatal("advance ran while op was in-flight")
		default:
		}
		// Clock has not moved while op holds.
		if sutClk.Now() != baseSUT {
			t.Fatal("sut clock advanced before release")
		}
		release()
		select {
		case <-advanceDone:
		case <-time.After(time.Second):
			t.Fatal("advance did not complete after release")
		}
	})
}
