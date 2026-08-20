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
// spent, and the budget refills as the clock moves. A run makes a fixed
// sequence of calls and cannot move anything, so a derived check would spend a
// handful of tokens out of a burst of ten and report success against a limiter
// that never refuses. The refusal is reached by declaring a second subject
// whose bucket is small and whose clock has already run.
func TestContractContract(t *testing.T) {
	t.Parallel()

	ratelimittest.RunContract(t,
		ratelimittest.ContractHarness[*ratelimittest.InMemory]{
			Name: "in-memory",
			New: func() *ratelimittest.InMemory {
				return ratelimittest.NewInMemory(clock.NewTestClock(origin), 10, period)
			},
		},
		ratelimittest.ContractHarness[*ratelimittest.InMemory]{
			Name: "in-memory, one token on a clock already moved",
			New: func() *ratelimittest.InMemory {
				clk := clock.NewTestClock(origin)
				s := ratelimittest.NewInMemory(clk, 1, period)
				clk.Advance(2 * period)
				return s
			},
		},
		ratelimittest.ContractChecks{
			{
				Method: "Run",
				Name:   "refuses-a-caller-with-nothing-left",
				Claim:  "Run refuses a caller with nothing left",
				Run: func(tb testing.TB, s ratelimit.Contract, fx ratelimittest.ContractFixture) {
					tb.Helper()
					// Spend until refused, which both subjects reach: the
					// generous one after its burst of ten, the spent one on
					// its first call. What is asserted is that the refusal
					// comes, and that it says the rate was the reason.
					for range 11 {
						err := s.Run(tb.Context(), fx.Key())
						if err == nil {
							continue
						}
						testkit.ErrorIs(tb, err, ratelimittest.ErrLimited,
							"a refusal says the rate was the reason")
						return
					}
					tb.Fatalf("a limiter that never refuses bounds nothing")
				},
			},
		},
	)
}

// Dropping a check is written against the typed index rather than a string, so
// a check that is renamed or stops being emitted breaks this compile instead of
// silently declining nothing.
func TestContractContractWithoutSmoke(t *testing.T) {
	t.Parallel()

	ratelimittest.RunContract(t,
		ratelimittest.ContractHarness[*ratelimittest.InMemory]{
			Name: "in-memory",
			New: func() *ratelimittest.InMemory {
				return ratelimittest.NewInMemory(clock.NewTestClock(origin), 10, period)
			},
		},
		ratelimittest.ContractSuite.Without(ratelimittest.ContractSuite.Checks.Run.Smoke()),
	)
}
