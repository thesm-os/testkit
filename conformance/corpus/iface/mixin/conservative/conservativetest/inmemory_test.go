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
		conservativetest.MixedOnTotal("holds the conserved sum through a transfer", func(
			tb testing.TB, subject conservative.Mixed,
		) {
			tb.Helper()
			// The suite seeds through Apply, and Apply is a transfer: the
			// conserved sum must still read as it did at birth — a non-zero
			// total is quantity minted from nothing, the mixin's violation.
			got, err := subject.Total(tb.Context())
			testkit.NoError(tb, err, "the total is readable")
			testkit.Equal(tb, got, 0, "and the transfer conserved it")
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
