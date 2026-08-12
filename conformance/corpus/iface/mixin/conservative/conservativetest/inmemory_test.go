// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package conservativetest_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/conservative"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/conservative/conservativetest"
)

// The generated contract, run against the in-memory subject.
func TestMixedContract(t *testing.T) {
	t.Parallel()

	conservativetest.AssertMixedContract(t,
		conservativetest.MixedModel(),
		conservativetest.MixedSubject("in-memory", func() conservative.Mixed {
			return conservativetest.NewInMemory()
		}),
		conservativetest.MixedOnTotal("reports what Apply folded in", func(
			tb testing.TB, subject conservative.Mixed,
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

	conservativetest.AssertMixedContract(t,
		conservativetest.MixedSubject("in-memory", func() conservative.Mixed {
			return conservativetest.NewInMemory()
		}),
		conservativetest.MixedWithoutDouble(),
	)
}
