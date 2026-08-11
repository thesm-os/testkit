// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package ratelimittest_test

import (
	"testing"
	"time"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/clock"
	ratelimit "go.thesmos.sh/testkit/conformance/corpus/iface/contract/rate-limit"
	"go.thesmos.sh/testkit/conformance/corpus/iface/contract/rate-limit/ratelimittest"
)

// origin is where every test clock in this package starts.
//
// A fixed instant rather than time.Now, so a failure reproduces exactly and a
// test that happened to run across a leap second behaves like every other run.
var origin = time.Date(2026, time.August, 11, 0, 0, 0, 0, time.UTC)

// period is how long one token takes to earn back.
const period = 10 * time.Millisecond

// rate-limit is owned by no tier under ADR-0018, which the gate reports as a law
// to write rather than a check to invent.
//
// The claim needs controlled time — a limiter refuses only after the budget is
// spent, and the budget refills as the clock moves. The harness makes a fixed
// sequence of calls and cannot move anything, so a generated check would spend
// a handful of tokens out of a burst of ten and report success against a
// limiter that never refuses.
//
// The subject is built with the burst the directive names, so the run's calls
// sit inside it and the family that does apply — cancellation, deadline, a nil
// context — is exercised against a limiter behaving normally.
func TestContractContract(t *testing.T) {
	t.Parallel()

	ratelimittest.AssertContractContract(t,
		ratelimittest.ContractSubject("in-memory", func() ratelimit.Contract {
			return ratelimittest.NewInMemory(clock.NewTestClock(origin), 10, period)
		}),
		ratelimittest.ContractSubject("in-memory, one token on a clock already moved", func() ratelimit.Contract {
			// The refusal and the refill are both out of a generated check's
			// reach — the harness makes a fixed number of calls and holds no
			// clock — and both are within a factory's, which may hand back a
			// subject whose bucket is small and whose clock has already run.
			//
			// The seed spends the one token, so the check below meets a limiter
			// with nothing left. That is the state the whole classification is
			// about, and a factory is what puts a subject in it.
			clk := clock.NewTestClock(origin)
			s := ratelimittest.NewInMemory(clk, 1, period)
			clk.Advance(2 * period)
			return s
		}),
		ratelimittest.ContractOnRun("refuses a caller with nothing left", func(
			tb testing.TB, subject ratelimit.Contract, key string,
		) {
			tb.Helper()
			// True of the spent subject and vacuous for the generous one, which
			// is the shape a two-subject claim takes when only one of them can
			// be in the state under check.
			for range 10 {
				if err := subject.Run(tb.Context(), key); err != nil {
					testkit.ErrorIs(tb, err, ratelimittest.ErrLimited,
						"a refusal says the rate was the reason")
					return
				}
			}
			testkit.NoError(tb, subject.Run(tb.Context(), key),
				"a limiter inside its burst keeps admitting")
		}),
	)
}

// Declining the double is separate from dropping a check.
func TestContractContractWithoutTheDouble(t *testing.T) {
	t.Parallel()

	ratelimittest.AssertContractContract(t,
		ratelimittest.ContractSubject("in-memory", func() ratelimit.Contract {
			return ratelimittest.NewInMemory(clock.NewTestClock(origin), 10, period)
		}),
		ratelimittest.ContractWithout("Run/smoke"),
		ratelimittest.ContractWithoutDouble(),
	)
}
