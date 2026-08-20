// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package castest_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/conformance/corpus/iface/contract/cas"
	"go.thesmos.sh/testkit/conformance/corpus/iface/contract/cas/castest"
)

// cas is the model tier's under ADR-0018: `AUTO-CAS-ATOMIC-ONE-WINNER` states
// it, and stating it needs accumulated version state — a stale revision to
// present, which only a sequence of writes produces.
//
// `version=Version` is an opaque param naming a field rather than a callable,
// so there is no partner to call and no verdict to read. The suite tier gets
// the signature-derived family, and the row states the one thing a fixed
// sequence can: a fresh cell accepts version zero and refuses anything else.
func TestContractContract(t *testing.T) {
	t.Parallel()

	castest.RunContract(t,
		castest.ContractHarness[*castest.InMemory]{Name: "in-memory", New: castest.NewInMemory},
		castest.ContractChecks{
			{
				Method: "Put",
				Name:   "fresh-cell-takes-version-zero",
				Claim:  "Put takes version zero against a fresh cell and refuses any other",
				Run: func(tb testing.TB, s cas.Contract, fx castest.ContractFixture) {
					tb.Helper()
					// The derived fixture cannot know the cell's dialect, so
					// the row spells it: every check gets a fresh subject whose
					// cell is back at the start.
					first := fx.Value()
					first.Version = 0
					testkit.NoError(tb, s.Put(tb.Context(), first), "version zero lands on a fresh cell")

					stale := fx.Value()
					stale.Version = 0
					testkit.Error(tb, s.Put(tb.Context(), stale),
						"and the same version a second time is stale")
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

	castest.RunContract(t,
		castest.ContractHarness[*castest.InMemory]{Name: "in-memory", New: castest.NewInMemory},
		castest.ContractSuite.Without(castest.ContractSuite.Checks.Put.Smoke()),
	)
}
