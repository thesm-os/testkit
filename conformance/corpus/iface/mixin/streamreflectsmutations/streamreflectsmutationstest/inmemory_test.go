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
// call: two checks, both about not crashing. Everything the mixin is about
// happens while someone is mid-range.
func TestMixedContract(t *testing.T) {
	t.Parallel()

	sm.AssertMixedContract(t,
		sm.MixedModel(),
		sm.MixedSubject("in-memory", func() streamreflectsmutations.Mixed {
			return sm.NewInMemory()
		}),
		sm.MixedOnStream("yields an item added while it is running", func(
			tb testing.TB, subject streamreflectsmutations.Mixed,
		) {
			tb.Helper()
			// The mixin's whole claim, and one the signature cannot make:
			// Stream returns a function, so a check that only called it would
			// assert that a closure was built.
			testkit.NoError(tb, subject.Add(tb.Context(), "a"), "the first item is added")

			var seen []string
			for item, err := range subject.Stream(tb.Context()) {
				testkit.NoError(tb, err, "the sequence yields without failing")
				seen = append(seen, item)
				if len(seen) == 1 {
					testkit.NoError(tb, subject.Add(tb.Context(), "b"),
						"an item added mid-range is accepted")
				}
			}
			testkit.Contains(tb, seen, "b", "and the run that added it sees it")
		}),
		sm.MixedOnStream("stops when the consumer does", func(
			tb testing.TB, subject streamreflectsmutations.Mixed,
		) {
			tb.Helper()
			// A sequence that ignored the consumer's break would run to the end
			// of a collection the caller stopped caring about — which for a
			// stream over a live store is unbounded.
			for range 3 {
				testkit.NoError(tb, subject.Add(tb.Context(), "x"), "an item is added")
			}

			taken := 0
			for range subject.Stream(tb.Context()) {
				taken++
				break
			}
			testkit.Equal(tb, taken, 1, "the range stopped where the consumer did")
		}),
		sm.MixedOnStream("reports a cancelled context through the sequence", func(
			tb testing.TB, subject streamreflectsmutations.Mixed,
		) {
			tb.Helper()
			// The only place a cancellation can surface: the signature returns
			// no error, so the sequence's own error slot is what carries it.
			ctx, cancel := context.WithCancel(tb.Context())
			cancel()

			var errs []error
			for _, err := range subject.Stream(ctx) {
				errs = append(errs, err)
			}
			testkit.Len(tb, errs, 1, "a cancelled caller is answered once")
			testkit.ErrorIs(tb, errs[0], context.Canceled, "with the cancellation")
		}),
	)
}

// Declining the double is separate from dropping a check.
func TestMixedContractWithoutTheDouble(t *testing.T) {
	t.Parallel()

	sm.AssertMixedContract(t,
		sm.MixedSubject("in-memory", func() streamreflectsmutations.Mixed {
			return sm.NewInMemory()
		}),
		sm.MixedWithout("Add/smoke"),
		sm.MixedWithoutDouble(),
	)
}
