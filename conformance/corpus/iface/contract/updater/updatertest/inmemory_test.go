// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package updatertest_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/conformance/corpus/iface/contract/updater"
	"go.thesmos.sh/testkit/conformance/corpus/iface/contract/updater/updatertest"
)

// The generated contract, run against the in-memory subject.
//
// Nothing distinguishes an updater from an upserter in either signature —
// both are Put beside Get — so what the two contracts claim differently is
// the row's to state.
func TestContractContract(t *testing.T) {
	t.Parallel()

	updatertest.RunContract(t,
		updatertest.ContractHarness[*updatertest.InMemory]{Name: "in-memory", New: updatertest.NewInMemory},
		updatertest.ContractChecks{
			{
				Method: "Put",
				Name:   "replaces-rather-than-accumulates",
				Claim:  "Put replaces rather than accumulates",
				Run: func(tb testing.TB, s updater.Contract, fx updatertest.ContractFixture) {
					tb.Helper()
					// The row writes the key first: an update needs something
					// to update, and a fresh subject holds nothing.
					first := fx.Value()
					testkit.NoError(tb, s.Put(tb.Context(), first), "the first write lands")

					replacement := updater.Value{Key: first.Key, Body: first.Body + "-replaced"}
					testkit.NoError(tb, s.Put(tb.Context(), replacement), "the update lands")

					got, err := s.Get(tb.Context(), first.Key)
					testkit.NoError(tb, err, "and the key is still there")
					testkit.Equal(tb, got, replacement, "carrying the newer value")
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

	updatertest.RunContract(t,
		updatertest.ContractHarness[*updatertest.InMemory]{Name: "in-memory", New: updatertest.NewInMemory},
		updatertest.ContractSuite.Without(updatertest.ContractSuite.Checks.Put.Smoke()),
	)
}
