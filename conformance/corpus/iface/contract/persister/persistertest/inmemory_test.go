// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package persistertest_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/conformance/corpus/iface/contract/persister"
	"go.thesmos.sh/testkit/conformance/corpus/iface/contract/persister/persistertest"
)

// persister is the model tier's under ADR-0018: `AUTO-PERSISTER-RETRIEVABLE`
// states it, and it needs a reference implementation to compare against.
//
// The suite tier still earns the pairing. Put is classified writer, so Get's
// miss check knows what a miss means here — an input nothing wrote — and the
// row below states the other half: what a key that WAS written reads back as.
func TestContractContract(t *testing.T) {
	t.Parallel()

	persistertest.RunContract(t,
		persistertest.ContractHarness[*persistertest.InMemory]{Name: "in-memory", New: persistertest.NewInMemory},
		persistertest.ContractChecks{
			{
				Method: "Get",
				Name:   "reads-back-what-put-wrote",
				Claim:  "Get returns what Put wrote under that key",
				Run: func(tb testing.TB, s persister.Contract, fx persistertest.ContractFixture) {
					tb.Helper()
					written := fx.Value()
					testkit.NoError(tb, s.Put(tb.Context(), written), "the value is stored")

					got, err := s.Get(tb.Context(), written.Key)
					testkit.NoError(tb, err, "the written key is found")
					testkit.Equal(tb, got, written, "carrying what was filed under it")
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

	persistertest.RunContract(t,
		persistertest.ContractHarness[*persistertest.InMemory]{Name: "in-memory", New: persistertest.NewInMemory},
		persistertest.ContractSuite.Without(persistertest.ContractSuite.Checks.Put.Smoke()),
	)
}
