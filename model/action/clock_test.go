// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package action_test

import (
	"testing"
	"time"

	"pgregory.net/rapid"

	"go.thesmos.sh/testkit/clock"
	"go.thesmos.sh/testkit/model/action"
)

func TestAdvanceClock(t *testing.T) {
	t.Parallel()

	t.Run("advances both clocks by same amount", func(t *testing.T) {
		t.Parallel()
		origin := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
		sutClk := clock.NewTestClock(origin)
		refClk := clock.NewTestClock(origin)

		a := action.AdvanceClock[string]("AdvanceClock",
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

		a := action.AdvanceClock[string]("AdvanceClock",
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
		a := action.AdvanceClock[string]("AdvanceClock",
			func() (*clock.TestClock, *clock.TestClock) { return nil, nil },
			time.Hour,
		)
		rapid.Check(t, func(rt *rapid.T) {
			a.Run(rt, "sut", "ref") // must not panic
		})
	})
}
