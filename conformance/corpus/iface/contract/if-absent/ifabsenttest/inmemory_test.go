// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package ifabsenttest_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	ifabsent "go.thesmos.sh/testkit/conformance/corpus/iface/contract/if-absent"
	"go.thesmos.sh/testkit/conformance/corpus/iface/contract/if-absent/ifabsenttest"
)

// if-absent is the suite tier's under ADR-0018: no [engine/model/law] property
// states "a second write for one key is refused", and the claim needs nothing
// the tier cannot produce — one subject, two calls, the same value both times.
//
// The row writes the key itself rather than expecting the run to have written
// it. Nothing seeds a subject now but its own constructor, so a check that
// wants a key present has to put it there.
func TestContractContract(t *testing.T) {
	t.Parallel()

	ifabsenttest.RunContract(t,
		ifabsenttest.ContractHarness[*ifabsenttest.InMemory]{Name: "in-memory", New: ifabsenttest.NewInMemory},
		ifabsenttest.ContractChecks{
			{
				Method: "Put",
				Name:   "refuses-the-key",
				Claim:  "Put refuses the key rather than the call",
				Run: func(tb testing.TB, s ifabsent.Contract, fx ifabsenttest.ContractFixture) {
					tb.Helper()
					// A store refusing every write after the first passes the
					// generated check without holding a key at all, which is the
					// reading of "refused" the contract does not mean.
					testkit.NoError(tb, s.Put(tb.Context(), fx.Value()),
						"a key nothing holds is accepted")
					testkit.Error(tb, s.Put(tb.Context(), fx.Value()),
						"the same key a second time is refused")
					testkit.NoError(tb, s.Put(tb.Context(), fx.ValueOther()),
						"and another key nothing holds is still accepted")
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

	ifabsenttest.RunContract(t,
		ifabsenttest.ContractHarness[*ifabsenttest.InMemory]{Name: "in-memory", New: ifabsenttest.NewInMemory},
		ifabsenttest.ContractSuite.Without(ifabsenttest.ContractSuite.Checks.Put.Smoke()),
	)
}
