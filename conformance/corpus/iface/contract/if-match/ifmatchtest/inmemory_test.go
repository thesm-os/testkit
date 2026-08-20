// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package ifmatchtest_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	ifmatch "go.thesmos.sh/testkit/conformance/corpus/iface/contract/if-match"
	"go.thesmos.sh/testkit/conformance/corpus/iface/contract/if-match/ifmatchtest"
)

// if-match is the suite tier's under ADR-0018, and the generated check states
// agreement rather than refusal.
//
// "A non-matching value is refused" needs a value this subject's predicate
// rejects, and neither the directive nor the signature says which one that is —
// so a derived value the predicate happens to admit would turn the check into a
// demand that a correct implementation fail. What the pair can always be asked
// is whether they agree, and the rejecting direction is stated below where a
// value the predicate declines can be constructed.
func TestContractContract(t *testing.T) {
	t.Parallel()

	ifmatchtest.RunContract(t,
		ifmatchtest.ContractHarness[*ifmatchtest.InMemory]{Name: "in-memory", New: ifmatchtest.NewInMemory},
		ifmatchtest.ContractChecks{
			{
				Method: "Put",
				Name:   "refuses-a-declined-value",
				Claim:  "Put refuses what the predicate declines",
				Run: func(tb testing.TB, s ifmatch.Contract, fx ifmatchtest.ContractFixture) {
					tb.Helper()
					// The half the generated check cannot reach: both derived
					// values are admitted, so the run only ever exercises
					// "accepts what Match admits". A stale body is what the
					// contract exists to catch — and the row writes the key
					// first, because a fresh subject holds nothing to be stale
					// against.
					held := fx.Value()
					testkit.NoError(tb, s.Put(tb.Context(), held), "the key is written")

					stale := ifmatch.Value{Key: held.Key, Body: held.Body + "-stale"}

					allowed, err := s.Match(tb.Context(), stale)
					testkit.NoError(tb, err, "the predicate answers about a key the store holds")
					testkit.False(tb, allowed, "and declines a body the store does not hold")

					testkit.Error(tb, s.Put(tb.Context(), stale),
						"so the write conditional on it is refused")
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

	ifmatchtest.RunContract(t,
		ifmatchtest.ContractHarness[*ifmatchtest.InMemory]{Name: "in-memory", New: ifmatchtest.NewInMemory},
		ifmatchtest.ContractSuite.Without(ifmatchtest.ContractSuite.Checks.Put.Smoke()),
	)
}
