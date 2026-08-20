// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package leasetest_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/conformance/corpus/iface/contract/lease"
	"go.thesmos.sh/testkit/conformance/corpus/iface/contract/lease/leasetest"
)

// lease is the model tier's under ADR-0018: `AUTO-LEASE-DOUBLE-ACQUIRE-BLOCKS`
// and `AUTO-LEASE-RELEASED-ON-CANCEL` state it.
//
// What the suite tier states is the row below: a key this run took is a key it
// cannot take twice, and one nobody holds is still available. Both halves,
// because a breaker refusing every acquire passes the first alone.
func TestContractContract(t *testing.T) {
	t.Parallel()

	leasetest.RunContract(t,
		leasetest.ContractHarness[*leasetest.InMemory]{Name: "in-memory", New: leasetest.NewInMemory},
		leasetest.ContractChecks{
			{
				Method: "Acquire",
				Name:   "refuses-a-held-key",
				Claim:  "Acquire refuses a key it already holds",
				Run: func(tb testing.TB, s lease.Contract, fx leasetest.ContractFixture) {
					tb.Helper()
					testkit.NoError(tb, s.Acquire(tb.Context(), fx.Key()), "a free key is taken")
					testkit.ErrorIs(tb, s.Acquire(tb.Context(), fx.Key()), lease.ErrHeld,
						"a held lease is refused rather than granted twice")
					testkit.NoError(tb, s.Acquire(tb.Context(), fx.KeyOther()),
						"and a key nobody holds is still available")
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

	leasetest.RunContract(t,
		leasetest.ContractHarness[*leasetest.InMemory]{Name: "in-memory", New: leasetest.NewInMemory},
		leasetest.ContractSuite.Without(leasetest.ContractSuite.Checks.Acquire.Smoke()),
	)
}
