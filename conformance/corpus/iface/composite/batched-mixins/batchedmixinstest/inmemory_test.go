// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package batchedmixinstest_test

import (
	"fmt"
	"testing"

	"go.thesmos.sh/testkit"
	batchedmixins "go.thesmos.sh/testkit/conformance/corpus/iface/composite/batched-mixins"
	"go.thesmos.sh/testkit/conformance/corpus/iface/composite/batched-mixins/batchedmixinstest"
)

// batched-mixins carries six classifications across three methods and generates
// no check for any of them: `idempotent`, `cacheable`, `pure`, `bounded` and
// `readafterwrite` are the model tier's under ADR-0018, and `concurrent` and
// `sideeffect` name no partner to observe through.
//
// What the fixture exists for is the parsing — extra positionals are further
// mixin names, and parameters are permitted only with exactly one name — so
// what the suite adds here is the signature-derived family and the claims
// below, every one of which is statable through the interface and therefore a
// check rather than a test in this package.
func TestBatchedContract(t *testing.T) {
	t.Parallel()

	batchedmixinstest.AssertBatchedContract(t,
		batchedmixinstest.BatchedModel(),
		batchedmixinstest.BatchedSubject("in-memory", func() batchedmixins.Batched {
			return batchedmixinstest.NewInMemory()
		}),
		batchedmixinstest.BatchedOnRead("returns what the seed wrote", func(
			tb testing.TB, subject batchedmixins.Batched, key string,
		) {
			tb.Helper()
			got, err := subject.Read(tb.Context(), key)
			testkit.NoError(tb, err, "the seeded key is found")
			testkit.NotEqual(tb, got, "", "carrying what was written")
		}),
		batchedmixinstest.BatchedOnPut("leaves the store where the first write left it", func(
			tb testing.TB, subject batchedmixins.Batched, key, value string,
		) {
			tb.Helper()
			// `idempotent` is the model tier's, and this is its single-subject
			// shadow: the seed already wrote this pair, so the call under check
			// is the repeat.
			before, err := subject.List(tb.Context())
			testkit.NoError(tb, err, "the store can be listed")

			testkit.NoError(tb, subject.Put(tb.Context(), key, value), "the repeated write lands")

			after, err := subject.List(tb.Context())
			testkit.NoError(tb, err, "and can still be listed")
			testkit.Equal(tb, after, before, "unchanged")
		}),
		batchedmixinstest.BatchedOnList("agrees with itself", func(
			tb testing.TB, subject batchedmixins.Batched,
		) {
			tb.Helper()
			// `pure` says the answer depends on the state and nothing else, and
			// Go's map iteration is deliberately unordered — so a subject
			// returning keys in range order fails here and passes everywhere a
			// single call is compared against an expected set.
			first, err := subject.List(tb.Context())
			testkit.NoError(tb, err, "the first listing succeeds")
			second, err := subject.List(tb.Context())
			testkit.NoError(tb, err, "and so does the second")
			testkit.Equal(tb, second, first, "and the two agree")
		}),
		batchedmixinstest.BatchedOnList("is bounded by the declared limit", func(
			tb testing.TB, subject batchedmixins.Batched,
		) {
			tb.Helper()
			// `bounded limit=50` is the model tier's, and nothing binds the
			// directive to the constant the subject obeys. Writing past it here
			// is what makes the number in the source answerable at all.
			for i := range 60 {
				testkit.NoError(tb, subject.Put(tb.Context(), fmt.Sprintf("k%02d", i), "v"),
					"a write lands")
			}
			got, err := subject.List(tb.Context())
			testkit.NoError(tb, err, "the store can be listed")
			testkit.Len(tb, got, 50, "and the listing stops at the declared limit")
		}),
	)
}

// Declining the double is separate from dropping a check.
func TestBatchedContractWithoutTheDouble(t *testing.T) {
	t.Parallel()

	batchedmixinstest.AssertBatchedContract(t,
		batchedmixinstest.BatchedSubject("in-memory", func() batchedmixins.Batched {
			return batchedmixinstest.NewInMemory()
		}),
		batchedmixinstest.BatchedWithout("Put/smoke"),
		batchedmixinstest.BatchedWithoutDouble(),
	)
}
