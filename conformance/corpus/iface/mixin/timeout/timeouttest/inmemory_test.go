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

// `//testkit:mixin timeout duration=5s` names a budget, and stating it needs a
// clock the run controls — which is the model tier's seam, not this one's. The
// signature family is what the suite derives.
func TestMixedContract(t *testing.T) {
	t.Parallel()

	timeouttest.RunMixed(t,
		timeouttest.MixedHarness[*timeouttest.InMemory]{Name: "in-memory", New: timeouttest.NewInMemory},
		timeouttest.MixedChecks{
			{
				Method: "Slow",
				Name:   "answers-at-once-with-nothing-to-wait-for",
				Claim:  "Slow answers at once when it has nothing to wait for",
				Run: func(tb testing.TB, s timeout.Mixed, fx timeouttest.MixedFixture) {
					tb.Helper()
					// A subject with no delay never reaches its clock, which is
					// the path a budget check cannot exercise: that one measures
					// a subject built to spend, and this one is built to spend
					// nothing.
					testkit.NoError(tb, s.Slow(tb.Context(), fx.Key()),
						"a subject with no delay answers without waiting")
				},
			},
		},
	)
}

// Dropping a check is written against the typed index rather than a string, so
// a check that is renamed or stops being emitted breaks this compile instead of
// silently declining nothing.
func TestMixedContractWithoutSmoke(t *testing.T) {
	t.Parallel()

	timeouttest.RunMixed(t,
		timeouttest.MixedHarness[*timeouttest.InMemory]{Name: "in-memory", New: timeouttest.NewInMemory},
		timeouttest.MixedSuite.Without(timeouttest.MixedSuite.Checks.Slow.Smoke()),
	)
}

// The budget, measured on a clock the test advances rather than on the wall.
//
// This is what the clock buys. Measuring with time.Now would make the budget a
// claim about how loaded the machine is: a correct implementation fails on a
// busy box, and proving the check can fail means genuinely spending five
// seconds. Measured on a clock the test advances, a subject that consumes six
// seconds does so instantly and the comparison is exact.
//
// Driven here rather than through a run: a subject built to wait hangs every
// check made against it, so the state this needs is one only a test holding the
// subject on its own can put it in.
func TestBudgetIsMeasuredOnTheRunsClock(t *testing.T) {
	t.Parallel()

	origin := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	t.Run("a subject inside the budget answers", func(t *testing.T) {
		t.Parallel()
		clk := clock.NewTestClock(origin)
		subject := timeouttest.WithDelay(clk, 4*time.Second)

		done := make(chan error, 1)
		go func() { done <- subject.Slow(t.Context(), "k") }()

		clk.AwaitWaiters(1)
		clk.Advance(4 * time.Second)

		testkit.NoError(t, <-done, "four seconds is inside a five-second budget")
		testkit.Equal(t, clk.Now().Sub(origin), 4*time.Second,
			"and the elapsed time is the clock's, exactly")
	})
}

// A caller who gives up is not left waiting on a clock nobody will advance.
func TestSlowAbandonsItsWaitWhenTheCallerGivesUp(t *testing.T) {
	t.Parallel()

	clk := clock.NewTestClock(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
	subject := timeouttest.WithDelay(clk, time.Hour)

	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan error, 1)
	go func() { done <- subject.Slow(ctx, "k") }()

	// The wait is on the test clock, so nothing but the cancellation can end
	// it — which is what makes the assertion about the subject rather than
	// about how long the test was willing to sit there.
	clk.AwaitWaiters(1)
	cancel()

	testkit.ErrorIs(t, <-done, context.Canceled,
		"the wait ends where the caller did, not where the clock would have")
}
