// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package singleflighttest_test

import (
	"sync"
	"sync/atomic"
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/conformance/corpus/iface/contract/singleflight"
	"go.thesmos.sh/testkit/conformance/corpus/iface/contract/singleflight/singleflighttest"
)

// singleflight is the model tier's under ADR-0018:
// `AUTO-SINGLEFLIGHT-COALESCES` states it, and states it with real contention —
// the law launches its own concurrent callers around the compute it supplies.
//
// The check below is the deterministic complement: with a memoizing subject,
// one computation per key is a fact, not a race to observe, so a hand-driven
// pair of callers can assert the exact count the law bounds.
func TestContractContract(t *testing.T) {
	t.Parallel()

	singleflighttest.RunContract(
		t,
		singleflighttest.ContractHarness[*singleflighttest.InMemory]{
			Name: "in-memory",
			New:  singleflighttest.NewInMemory,
		},
		singleflighttest.ContractChecks{
			{
				Method: "Run",
				Name:   "computes-once-per-key",
				Claim:  "Run computes once per key and shares the answer",
				Run: func(tb testing.TB, s singleflight.Contract, fx singleflighttest.ContractFixture) {
					tb.Helper()
					// The compute is the row's, not the fixture's: counting the
					// calls is the whole claim, and a derived func literal has
					// no counter in it.
					var calls atomic.Int64
					compute := func() string {
						calls.Add(1)
						return "computed"
					}

					const callers = 4
					var wg sync.WaitGroup
					for range callers {
						wg.Go(func() {
							got, err := s.Run(tb.Context(), fx.Key(), compute)
							testkit.NoError(tb, err, "a caller is answered")
							testkit.Equal(tb, got, "computed", "with the shared answer")
						})
					}
					wg.Wait()

					testkit.Equal(tb, calls.Load(), int64(1),
						"four callers for one key ran the compute once")
				},
			},
		},
	)
}

// Dropping a check is written against the typed index rather than a string, so
// a check that is renamed or stops being emitted breaks this compile instead of
// silently declining nothing.
//
// Flights rather than Run: Run's compute argument admits no literal, so no
// check is derived for it at all and Checks has no Run member to name. The
// header says as much, and the row above is what covers the method instead.
func TestContractContractWithoutSmoke(t *testing.T) {
	t.Parallel()

	singleflighttest.RunContract(
		t,
		singleflighttest.ContractHarness[*singleflighttest.InMemory]{
			Name: "in-memory",
			New:  singleflighttest.NewInMemory,
		},
		singleflighttest.ContractSuite.Without(singleflighttest.ContractSuite.Checks.Flights.Smoke()),
	)
}
