// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package boundedtest_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/bounded"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/bounded/boundedtest"
)

// bounded is the model tier's — AUTO-AGGREGATOR-BOUNDED states it — so the
// suite generates the signature family alone.
//
// The declared ceiling reaches the subject rather than being restated by it:
// `//testkit:mixin bounded limit=5` is what the harness hands every
// constructor, so a subject built at some other capacity is one the law would
// measure against a limit it was never given.
func TestMixedContract(t *testing.T) {
	t.Parallel()

	boundedtest.RunMixed(t,
		boundedtest.MixedHarness[*boundedtest.InMemory]{Name: "in-memory", New: boundedtest.NewInMemory},
		boundedtest.MixedChecks{
			{
				Method: "List",
				Name:   "clamped-to-the-declared-bound",
				Claim:  "List is bounded by the capacity the declaration gave it",
				Run: func(tb testing.TB, s bounded.Mixed, fx boundedtest.MixedFixture) {
					tb.Helper()
					// One more than the bound, so the clamp has something to
					// clamp. A collection that grew without answering more is
					// exactly what the mixin claims.
					for range 7 {
						testkit.NoError(tb, s.Add(tb.Context(), fx.Item()), "an item is added")
					}

					got, err := s.List(tb.Context())
					testkit.NoError(tb, err, "the collection is readable")
					testkit.Len(tb, got, 5, "and answers no more than the declared bound")
				},
			},
		},
	)
}

// Dropping a check is written against the typed index rather than a string, so
// a check that is renamed or stops being emitted breaks this compile instead of
// silently declining nothing.
func TestMixedContractWithoutSmoke(t *testing.T) {
	t.Parallel()

	boundedtest.RunMixed(t,
		boundedtest.MixedHarness[*boundedtest.InMemory]{Name: "in-memory", New: boundedtest.NewInMemory},
		boundedtest.MixedSuite.Without(boundedtest.MixedSuite.Checks.List.Smoke()),
	)
}
