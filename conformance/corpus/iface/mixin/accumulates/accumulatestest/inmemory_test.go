// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package accumulatestest_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/accumulates"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/accumulates/accumulatestest"
)

// The effect axis's two positions in one file, split across two tiers.
//
// Add/accumulates is derived and says what one subject and a fixed sequence
// settle: the second call is taken rather than refused. That is the half a
// coalescing store gets wrong first, and it is all the generated check can
// say — the mixin names no observer, and nothing in Add's signature points at
// Total.
//
// The compounding itself is the row below. It needs something to read the
// effect through, which is the same reason `idempotent`'s real law is the
// model tier's rather than its repeat probe.
func TestMixedContract(t *testing.T) {
	t.Parallel()

	accumulatestest.RunMixed(t,
		accumulatestest.MixedHarness[*accumulatestest.InMemory]{Name: "in-memory", New: accumulatestest.NewInMemory},
		accumulatestest.MixedChecks{
			{
				Method: "Total",
				Name:   "two-adds-compound",
				Claim:  "Total reports the sum of the additions rather than the last one",
				Run: func(tb testing.TB, s accumulates.Mixed, fx accumulatestest.MixedFixture) {
					tb.Helper()
					// The whole of the mixin, and the assertion that separates
					// it from idempotent over the same two methods: a store
					// that replaced would answer the amount once.
					testkit.NoError(tb, s.Add(tb.Context(), fx.Key(), 3), "the first addition lands")
					testkit.NoError(tb, s.Add(tb.Context(), fx.Key(), 3), "and so does the second")

					got, err := s.Total(tb.Context(), fx.Key())
					testkit.NoError(tb, err, "the total is readable")
					testkit.Equal(tb, got, 6, "and counts both additions")
				},
			},
			{
				Method: "Total",
				Name:   "unadded-key-totals-zero",
				Claim:  "Total reports zero for a key nothing added to",
				Run: func(tb testing.TB, s accumulates.Mixed, fx accumulatestest.MixedFixture) {
					tb.Helper()
					// The sum of no additions, which is an answer rather than a
					// failure — and what makes Total/miss's zero the right
					// claim rather than a sentinel.
					got, err := s.Total(tb.Context(), fx.KeyOther())
					testkit.NoError(tb, err, "an unadded key is not an error")
					testkit.Equal(tb, got, 0, "it simply has nothing summed")
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

	accumulatestest.RunMixed(t,
		accumulatestest.MixedHarness[*accumulatestest.InMemory]{Name: "in-memory", New: accumulatestest.NewInMemory},
		accumulatestest.MixedSuite.Without(accumulatestest.MixedSuite.Checks.Add.Smoke()),
	)
}
