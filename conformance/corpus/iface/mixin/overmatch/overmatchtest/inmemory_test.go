// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package overmatchtest_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/overmatch"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/overmatch/overmatchtest"
)

// The generated contract, run against the in-memory subject.
//
// Nothing in either signature pairs Add with Items, so what a drain owes the
// appends before it is the row's claim.
func TestMixedContract(t *testing.T) {
	t.Parallel()

	overmatchtest.RunMixed(t,
		overmatchtest.MixedHarness[*overmatchtest.InMemory]{Name: "in-memory", New: overmatchtest.NewInMemory},
		overmatchtest.MixedChecks{
			{
				Method: "Items",
				Name:   "yields-each-append-once",
				Claim:  "Items yields what Add put in, once each",
				Run: func(tb testing.TB, s overmatch.Mixed, fx overmatchtest.MixedFixture) {
					tb.Helper()
					// The same key twice is what makes the claim testable: a
					// drain that yielded it twice would be reporting its input
					// rather than its contents.
					testkit.NoError(tb, s.Add(tb.Context(), overmatch.Value{Key: "zz", Body: "last"}),
						"an element is accepted")
					testkit.NoError(tb, s.Add(tb.Context(), overmatch.Value{Key: "zz", Body: "last"}),
						"and so is the same key again")

					// A second element, deliberately out of key order: with one
					// element the drain's ordering is unobservable, and a
					// subject that returned map order would pass.
					testkit.NoError(tb, s.Add(tb.Context(), overmatch.Value{Key: "aa", Body: "first"}),
						"and a second that sorts ahead of it")

					got, err := s.Items(tb.Context())
					testkit.NoError(tb, err, "the drain succeeds")
					testkit.Equal(tb, len(got), 2, "the repeated key is one element")
					testkit.Equal(tb, got[0].Key, "aa", "and the drain is ordered rather than arbitrary")
				},
			},
		},
	)
}
