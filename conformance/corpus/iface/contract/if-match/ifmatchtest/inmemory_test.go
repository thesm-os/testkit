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

	ifmatchtest.AssertContractContract(t,
		ifmatchtest.ContractSubject("in-memory", func() ifmatch.Contract {
			return ifmatchtest.NewInMemory()
		}),
		ifmatchtest.ContractOnPut("refuses what the predicate declines", func(
			tb testing.TB, subject ifmatch.Contract, v ifmatch.Value,
		) {
			tb.Helper()
			// The half the generated check cannot reach: both derived values
			// are admitted, so the run only ever exercises "accepts what Match
			// admits". A stale body is what the contract exists to catch.
			stale := ifmatch.Value{Key: v.Key, Body: v.Body + "-stale"}

			allowed, err := subject.Match(tb.Context(), stale)
			testkit.NoError(tb, err, "the predicate answers about a key the seed wrote")
			testkit.False(tb, allowed, "and declines a body the store does not hold")

			testkit.Error(tb, subject.Put(tb.Context(), stale),
				"so the write conditional on it is refused")
		}),
	)
}

// Declining the double is separate from dropping a check.
func TestContractContractWithoutTheDouble(t *testing.T) {
	t.Parallel()

	ifmatchtest.AssertContractContract(t,
		ifmatchtest.ContractSubject("in-memory", func() ifmatch.Contract {
			return ifmatchtest.NewInMemory()
		}),
		ifmatchtest.ContractWithout("Put/smoke"),
		ifmatchtest.ContractWithoutDouble(),
	)
}
