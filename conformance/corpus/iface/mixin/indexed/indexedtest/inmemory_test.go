// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package indexedtest_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/indexed"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/indexed/indexedtest"
)

// The generated contract, run against the in-memory subject.
func TestRankedContract(t *testing.T) {
	t.Parallel()

	indexedtest.AssertRankedContract(t,
		indexedtest.RankedModel(),
		indexedtest.RankedSubject("in-memory", func() indexed.Ranked {
			return indexedtest.NewInMemory()
		}),
		indexedtest.RankedOnAt("misses a position the collection does not hold", func(
			tb testing.TB, subject indexed.Ranked, _ int,
		) {
			tb.Helper()
			// The claim the mixin exists to make checkable: a position is
			// only meaningful against the size Len reports. Until the
			// derivation draws inside that size, this is the check that
			// states it — by hand, on the one subject that can.
			n, err := subject.Len(tb.Context())
			testkit.NoError(tb, err, "the size is readable")

			_, err = subject.At(tb.Context(), n)
			testkit.ErrorIs(tb, err, indexedtest.ErrOutOfRange,
				"one past the last element is not a position")

			if n > 0 {
				_, err = subject.At(tb.Context(), n-1)
				testkit.NoError(tb, err, "the last element is")
			}
		}),
	)
}

// Declining the double is separate from dropping a check.
func TestRankedContractWithoutTheDouble(t *testing.T) {
	t.Parallel()

	indexedtest.AssertRankedContract(t,
		indexedtest.RankedSubject("in-memory", func() indexed.Ranked {
			return indexedtest.NewInMemory()
		}),
		indexedtest.RankedWithoutDouble(),
	)
}

// The saturation prover: every bound law must be able to fail as itself,
// a defect worn on its own methods reddening the run by name.
func TestRankedSaturation(t *testing.T) {
	t.Parallel()
	indexedtest.RankedModelSaturation(t, func() indexed.Ranked {
		return indexedtest.NewInMemory()
	})
}
