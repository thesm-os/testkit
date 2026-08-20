// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package streamreflectsmutationstest_test

import (
	"context"
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/streamreflectsmutations"
	sm "go.thesmos.sh/testkit/conformance/corpus/iface/mixin/streamreflectsmutations/streamreflectsmutationstest"
)

// streamreflectsmutations is the model tier's — AUTO-STREAM-REFLECTS-MUTATIONS
// states it — so the suite generates the signature family alone, even though
// eidos now lets the mixin name its mutator through `mutate=Add`.
//
// Stream returns a function, so what the signature can promise ends at the
// call: one check, about not crashing. Everything the mixin is about happens
// while someone is mid-range.
func TestMixedContract(t *testing.T) {
	t.Parallel()

	sm.RunMixed(t,
		sm.MixedHarness[*sm.InMemory]{Name: "in-memory", New: sm.NewInMemory},
		sm.MixedChecks{
			{
				Method: "Stream",
				Name:   "yields-an-item-added-mid-range",
				Claim:  "Stream yields an item added while it is running",
				Run: func(tb testing.TB, s streamreflectsmutations.Mixed, fx sm.MixedFixture) {
					tb.Helper()
					// The mixin's whole claim, and one the signature cannot
					// make: Stream returns a function, so a check that only
					// called it would assert that a closure was built.
					testkit.NoError(tb, s.Add(tb.Context(), "a"), "the first item is added")

					var seen []string
					for item, err := range s.Stream(tb.Context()) {
						testkit.NoError(tb, err, "the sequence yields without failing")
						seen = append(seen, item)
						if len(seen) == 1 {
							testkit.NoError(tb, s.Add(tb.Context(), "b"),
								"an item added mid-range is accepted")
						}
					}
					testkit.Contains(tb, seen, "b", "and the run that added it sees it")
				},
			},
			{
				Method: "Stream",
				Name:   "stops-when-the-consumer-does",
				Claim:  "Stream stops when the consumer does",
				Run: func(tb testing.TB, s streamreflectsmutations.Mixed, fx sm.MixedFixture) {
					tb.Helper()
					// A sequence that ignored the consumer's break would run to
					// the end of a collection the caller stopped caring about —
					// which for a stream over a live store is unbounded.
					for range 3 {
						testkit.NoError(tb, s.Add(tb.Context(), fx.Item()), "an item is added")
					}

					taken := 0
					for range s.Stream(tb.Context()) {
						taken++
						break
					}
					testkit.Equal(tb, taken, 1, "the range stopped where the consumer did")
				},
			},
			{
				Method: "Stream",
				Name:   "cancellation-through-the-sequence",
				Claim:  "Stream reports a cancelled context through the sequence",
				Run: func(tb testing.TB, s streamreflectsmutations.Mixed, fx sm.MixedFixture) {
					tb.Helper()
					// The only place a cancellation can surface: the signature
					// returns no error, so the sequence's own error slot is
					// what carries it.
					ctx, cancel := context.WithCancel(tb.Context())
					cancel()

					var errs []error
					for _, err := range s.Stream(ctx) {
						errs = append(errs, err)
					}
					testkit.Len(tb, errs, 1, "a cancelled caller is answered once")
					testkit.ErrorIs(tb, errs[0], context.Canceled, "with the cancellation")
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

	sm.RunMixed(t,
		sm.MixedHarness[*sm.InMemory]{Name: "in-memory", New: sm.NewInMemory},
		sm.MixedSuite.Without(sm.MixedSuite.Checks.Add.Smoke()),
	)
}
