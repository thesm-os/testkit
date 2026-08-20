// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package upsertertest_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/conformance/corpus/iface/contract/upserter"
	"go.thesmos.sh/testkit/conformance/corpus/iface/contract/upserter/upsertertest"
)

// The generated contract, run against the in-memory subject.
//
// The pair to updater: identical signatures, and the difference between the
// two contracts is what the row states — a repeated write of the SAME value
// leaves the store where the first write left it.
func TestContractContract(t *testing.T) {
	t.Parallel()

	upsertertest.RunContract(t,
		upsertertest.ContractHarness[*upsertertest.InMemory]{Name: "in-memory", New: upsertertest.NewInMemory},
		upsertertest.ContractChecks{
			{
				Method: "Put",
				Name:   "repeat-write-is-the-same-key",
				Claim:  "Put writes the same key a second time rather than a new one",
				Run: func(tb testing.TB, s upserter.Contract, fx upsertertest.ContractFixture) {
					tb.Helper()
					v := fx.Value()
					testkit.NoError(tb, s.Put(tb.Context(), v), "the first write lands")
					testkit.NoError(tb, s.Put(tb.Context(), v), "the repeated write lands")

					got, err := s.Get(tb.Context(), v.Key)
					testkit.NoError(tb, err, "and the key is still there")
					testkit.Equal(tb, got, v, "carrying what it carried before")
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

	upsertertest.RunContract(t,
		upsertertest.ContractHarness[*upsertertest.InMemory]{Name: "in-memory", New: upsertertest.NewInMemory},
		upsertertest.ContractSuite.Without(upsertertest.ContractSuite.Checks.Put.Smoke()),
	)
}
