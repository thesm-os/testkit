// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package commutativetest_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/commutative"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/commutative/commutativetest"
)

// The generated contract, run against the in-memory subject.
func TestMixedContract(t *testing.T) {
	t.Parallel()

	commutativetest.AssertMixedContract(t,
		commutativetest.MixedSubject("in-memory", func() commutative.Mixed {
			return commutativetest.NewInMemory()
		}),
		commutativetest.MixedOnTotal("reports what Apply folded in", func(
			tb testing.TB, subject commutative.Mixed,
		) {
			tb.Helper()
			// The suite seeds through Apply, so the fold is non-empty here —
			// a total of zero would be a fold that dropped its input.
			got, err := subject.Total(tb.Context())
			testkit.NoError(tb, err, "the total is readable")
			testkit.NotEqual(tb, got, 0, "and reflects the seeded delta")
		}),
	)
}

// Declining the double is separate from dropping a check.
func TestMixedContractWithoutTheDouble(t *testing.T) {
	t.Parallel()

	commutativetest.AssertMixedContract(t,
		commutativetest.MixedSubject("in-memory", func() commutative.Mixed {
			return commutativetest.NewInMemory()
		}),
		commutativetest.MixedWithoutDouble(),
	)
}
