// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package timeouttest_test

import (
	"context"
	"testing"
	"time"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/clock"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/timeout"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/timeout/timeouttest"
)

// The only generated check that reads a value out of the source, and the only
// one that reads time.
//
// `//testkit:mixin timeout duration=5s` is the budget, and the check is gated on
// the parameter rather than on the mixin: "within a budget" is not a statement
// until one is named, so a bare `//testkit:mixin timeout` generates nothing.
func TestMixedContract(t *testing.T) {
	t.Parallel()

	timeouttest.AssertMixedContract(t,
		timeouttest.MixedSubject("in-memory", func() timeout.Mixed {
			return timeouttest.NewInMemory()
		}),
	)
}

// Both verdicts, settled exactly and costing no wall-clock time at all.
//
// This is what the clock buys. Measuring with time.Now would make the budget a
// claim about how loaded the machine is: a correct implementation fails on a
// busy box, and proving the check can fail means genuinely spending five
// seconds. Measured on a clock the test advances, a subject that consumes six
// seconds does so instantly and the comparison is exact.
func TestBudgetIsMeasuredOnTheRunsClock(t *testing.T) {
	t.Parallel()

	origin := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	t.Run("passes for a subject inside the budget", func(t *testing.T) {
		t.Parallel()
		clk := clock.NewTestClock(origin)
		f := testkit.NewFailableTB()

		go func() {
			clk.AwaitWaiters(1)
			clk.Advance(4 * time.Second)
		}()
		timeouttest.AssertMixedSlowCompletesInBudget(f, clk,
			timeouttest.WithDelay(clk, 4*time.Second), "key")

		testkit.False(t, f.Failed(), "four seconds is inside a five-second budget")
	})

	t.Run("fails for a subject over it", func(t *testing.T) {
		t.Parallel()
		clk := clock.NewTestClock(origin)
		f := testkit.NewFailableTB()

		go func() {
			clk.AwaitWaiters(1)
			clk.Advance(6 * time.Second)
		}()
		timeouttest.AssertMixedSlowCompletesInBudget(f, clk,
			timeouttest.WithDelay(clk, 6*time.Second), "key")

		testkit.True(t, f.Failed(), "six seconds is over a five-second budget")
	})
}

// A delayed call abandons its wait when the caller gives up, rather than
// finishing and reporting success to nobody.
func TestSlowAbandonsItsDelayWhenTheContextIsDone(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(t.Context())
	clk := clock.NewTestClock(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))

	go func() {
		clk.AwaitWaiters(1)
		cancel()
	}()

	err := timeouttest.WithDelay(clk, time.Hour).Slow(ctx, "key")
	testkit.ErrorIs(t, err, context.Canceled,
		"a caller that gave up is told so rather than waited out")
}

// Declining the double is separate from dropping a check.
func TestMixedContractWithoutTheDouble(t *testing.T) {
	t.Parallel()

	timeouttest.AssertMixedContract(t,
		timeouttest.MixedSubject("in-memory", func() timeout.Mixed {
			return timeouttest.NewInMemory()
		}),
		timeouttest.MixedWithout("Slow/smoke"),
		timeouttest.MixedWithoutDouble(),
	)
}
