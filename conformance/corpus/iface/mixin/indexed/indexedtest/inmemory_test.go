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

	indexedtest.RunRanked(t,
		indexedtest.RankedHarness[*indexedtest.InMemory]{Name: "in-memory", New: indexedtest.NewInMemory},
		indexedtest.RankedChecks{
			{
				Method: "At",
				Name:   "one-past-the-end-is-not-a-position",
				Claim:  "At misses a position the collection does not hold",
				Run: func(tb testing.TB, s indexed.Ranked, fx indexedtest.RankedFixture) {
					tb.Helper()
					// The claim the mixin exists to make checkable: a position
					// is only meaningful against the size Len reports. The row
					// adds first, so the boundary it walks to has an element
					// on the inside of it as well as the outside.
					testkit.NoError(tb, s.Add(tb.Context(), fx.Value()), "an element is added")

					n, err := s.Len(tb.Context())
					testkit.NoError(tb, err, "the size is readable")
					testkit.Equal(tb, n, 1, "and counts what was added")

					_, err = s.At(tb.Context(), n)
					testkit.ErrorIs(tb, err, indexedtest.ErrOutOfRange,
						"one past the last element is not a position")

					_, err = s.At(tb.Context(), n-1)
					testkit.NoError(tb, err, "the last element is")
				},
			},
		},
	)
}
