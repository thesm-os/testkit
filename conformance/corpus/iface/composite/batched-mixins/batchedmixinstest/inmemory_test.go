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
// what the suite adds here is the signature-derived family and the rows below,
// every one of which is statable through the interface.
//
// The declared bound reaches the subject rather than being restated by it:
// `bounded limit=50` is what the harness hands every constructor.
func TestBatchedContract(t *testing.T) {
	t.Parallel()

	batchedmixinstest.RunBatched(
		t,
		batchedmixinstest.BatchedHarness[*batchedmixinstest.InMemory]{
			Name: "in-memory",
			New:  batchedmixinstest.NewInMemory,
		},
		batchedmixinstest.BatchedChecks{
			{
				Method: "Read",
				Name:   "reads-back-what-put-wrote",
				Claim:  "Read returns what Put wrote",
				Run: func(tb testing.TB, s batchedmixins.Batched, fx batchedmixinstest.BatchedFixture) {
					tb.Helper()
					testkit.NoError(tb, s.Put(tb.Context(), fx.Key(), fx.Value()), "the key is written")

					got, err := s.Read(tb.Context(), fx.Key())
					testkit.NoError(tb, err, "the written key is found")
					testkit.Equal(tb, got, fx.Value(), "carrying what was written")
				},
			},
			{
				Method: "Put",
				Name:   "repeat-write-changes-nothing",
				Claim:  "Put leaves the store where the first write left it",
				Run: func(tb testing.TB, s batchedmixins.Batched, fx batchedmixinstest.BatchedFixture) {
					tb.Helper()
					// `idempotent` is the model tier's, and this is its
					// single-subject shadow: the row writes once, then repeats.
					testkit.NoError(tb, s.Put(tb.Context(), fx.Key(), fx.Value()), "the first write lands")

					before, err := s.List(tb.Context())
					testkit.NoError(tb, err, "the store can be listed")

					testkit.NoError(tb, s.Put(tb.Context(), fx.Key(), fx.Value()), "the repeated write lands")

					after, err := s.List(tb.Context())
					testkit.NoError(tb, err, "and can still be listed")
					testkit.Equal(tb, after, before, "unchanged")
				},
			},
			{
				Method: "List",
				Name:   "agrees-with-itself",
				Claim:  "List agrees with itself",
				Run: func(tb testing.TB, s batchedmixins.Batched, fx batchedmixinstest.BatchedFixture) {
					tb.Helper()
					// `pure` says the answer depends on the state and nothing
					// else, and Go's map iteration is deliberately unordered —
					// so a subject returning keys in range order fails here and
					// passes everywhere a single call is compared against an
					// expected set.
					testkit.NoError(tb, s.Put(tb.Context(), fx.Key(), fx.Value()), "a key is written")
					testkit.NoError(tb, s.Put(tb.Context(), fx.KeyOther(), fx.ValueOther()),
						"and a second, so order is observable")

					first, err := s.List(tb.Context())
					testkit.NoError(tb, err, "the first listing succeeds")
					second, err := s.List(tb.Context())
					testkit.NoError(tb, err, "and so does the second")
					testkit.Equal(tb, second, first, "and the two agree")
				},
			},
			{
				Method: "List",
				Name:   "bounded-by-the-declared-limit",
				Claim:  "List is bounded by the capacity the declaration gave it",
				Run: func(tb testing.TB, s batchedmixins.Batched, fx batchedmixinstest.BatchedFixture) {
					tb.Helper()
					// `bounded limit=50` reaches the subject through the
					// harness, so writing past it here is what makes the number
					// in the source answerable at all.
					for i := range 60 {
						testkit.NoError(tb, s.Put(tb.Context(), fmt.Sprintf("k%02d", i), "v"),
							"a write lands")
					}
					got, err := s.List(tb.Context())
					testkit.NoError(tb, err, "the store can be listed")
					testkit.Len(tb, got, 50, "and the listing stops at the declared limit")
				},
			},
		},
	)
}

// Dropping a check is written against the typed index rather than a string, so
// a check that is renamed or stops being emitted breaks this compile instead of
// silently declining nothing.
func TestBatchedContractWithoutSmoke(t *testing.T) {
	t.Parallel()

	batchedmixinstest.RunBatched(
		t,
		batchedmixinstest.BatchedHarness[*batchedmixinstest.InMemory]{
			Name: "in-memory",
			New:  batchedmixinstest.NewInMemory,
		},
		batchedmixinstest.BatchedSuite.Without(batchedmixinstest.BatchedSuite.Checks.Put.Smoke()),
	)
}
