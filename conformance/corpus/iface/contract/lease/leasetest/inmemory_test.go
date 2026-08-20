// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// `lease` is the model tier's: that acquires and releases balance over a whole
// history is what its laws state. The row below is what one key settles — a
// held lease is refused, and another key is unaffected.
package leasetest_test

import (
	"context"
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/conformance/corpus/iface/contract/lease"
	"go.thesmos.sh/testkit/conformance/corpus/iface/contract/lease/leasetest"
)

// TestContractContract runs the generated checks and this package's own.
func TestContractContract(t *testing.T) {
	t.Parallel()

	leasetest.RunContract(t, inMemory("in-memory"), contractChecks)
}

// TestContractContractWithoutSmoke drops a check through the typed index rather
// than a string, so a check that is renamed or stops being emitted breaks this
// compile instead of silently declining nothing.
func TestContractContractWithoutSmoke(t *testing.T) {
	t.Parallel()

	leasetest.RunContract(t,
		inMemory("in-memory"),
		leasetest.ContractSuite.Without(leasetest.ContractSuite.Checks.Acquire.Smoke()),
	)
}

// TestContractChecksCanFail drives the row against its planted defect.
func TestContractChecksCanFail(t *testing.T) {
	t.Parallel()

	leasetest.ProveContract(t, contractChecks)
}

// --- Harnesses ---------------------------------------------------------------

func inMemory(name string) leasetest.ContractHarness[*leasetest.InMemory] {
	return leasetest.ContractHarness[*leasetest.InMemory]{
		Name: name, New: leasetest.NewInMemory,
	}
}

// --- The checks: claims, bodies and defects, by name --------------------------

var contractChecks = leasetest.ContractChecks{
	{
		Method: "Acquire", Name: "refuses-a-held-key",
		Claim: "Acquire refuses a key it already holds",
		Run:   refusesAHeldKey,
		ProvenBy: leasetest.BrokenContract(
			"a lease that grants every key it is asked for", newGrantsEverything,
		),
		ProvenReason: "refused rather than granted twice",
	},
}

// --- Bodies -------------------------------------------------------------------

func refusesAHeldKey(tb testing.TB, s lease.Contract, fx leasetest.ContractFixture) {
	tb.Helper()
	testkit.NoError(tb, s.Acquire(tb.Context(), fx.Key()), "a free key is taken")
	testkit.ErrorIs(tb, s.Acquire(tb.Context(), fx.Key()), lease.ErrHeld,
		"a held lease is refused rather than granted twice")
	testkit.NoError(tb, s.Acquire(tb.Context(), fx.KeyOther()),
		"and a key nobody holds is still available")
}

// --- Planted defects ----------------------------------------------------------

// grantsEverything hands out every lease it is asked for, which is mutual
// exclusion with no state behind it. Every generated check passes: each meets a
// fresh subject and takes one lease, which this grants correctly.
type grantsEverything struct{}

func newGrantsEverything() grantsEverything { return grantsEverything{} }

func (grantsEverything) Acquire(context.Context, string) error { return nil }

func (grantsEverything) Release(context.Context, string) error { return nil }
