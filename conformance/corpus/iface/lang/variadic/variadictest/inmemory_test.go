// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package variadictest_test

import (
	"context"
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/conformance/corpus/iface/lang/variadic"
	"go.thesmos.sh/testkit/conformance/corpus/iface/lang/variadic/variadictest"
)

// A variadic parameter is one fixture field, so a generated check witnesses one
// element. Everything about *several* is the author's to state, which is what
// the two checks below are.
func TestFinderContract(t *testing.T) {
	t.Parallel()

	// The values the run itself uses, read rather than replaced: nothing here
	// passes FinderWithFixture, so the derivation stands and a seed built from
	// it writes what the checks are handed.
	fixture := variadictest.DefaultFinderFixture()

	variadictest.AssertFinderContract(t,
		variadictest.FinderSubject("in-memory", func() variadic.Finder {
			return variadictest.NewInMemory()
		}),
		variadictest.FinderSeed(func(_ context.Context, subject variadic.Finder) error {
			// A seed may reach for the concrete subject, because it runs before
			// the double wraps it and sees what the factory made. A check may
			// not: it runs against every subject the suite is given.
			s := subject.(*variadictest.InMemory)
			s.Put(fixture.Keys, "first")
			s.Put(fixture.KeysOther, "second")
			return nil
		}),
		variadictest.FinderOnFind("returns one result per key it holds, in order", func(
			tb testing.TB, subject variadic.Finder, keys string,
		) {
			tb.Helper()
			got, err := subject.Find(tb.Context(), fixture.Keys, fixture.KeysOther)
			testkit.NoError(tb, err, "a lookup of two held keys succeeds")
			testkit.Equal(tb, got, []string{"first", "second"},
				"several keys are answered in the order asked")
		}),
		variadictest.FinderOnFind("reports a lookup with nothing to look up", func(
			tb testing.TB, subject variadic.Finder, keys string,
		) {
			tb.Helper()
			_, err := subject.Find(tb.Context())
			testkit.ErrorIs(tb, err, variadictest.ErrNoKeys,
				"the empty variadic form is the one a derived check cannot reach")
		}),
		// This subject never errors for a non-empty key list — an absent key is
		// simply absent from the result — so the miss the check needs does not
		// exist. Dropping it by name is what keeps the other nine running; the
		// alternative is a consumer who abandons the suite over one check.
		variadictest.FinderWithout(
			"Find/an error carries the zero value",
			"FindWithLimit/an error carries the zero value",
		),
		variadictest.FinderOnFindWithLimit("truncates to the limit", func(
			tb testing.TB, subject variadic.Finder, limit int, keys string,
		) {
			tb.Helper()
			got, err := subject.FindWithLimit(tb.Context(), 1, fixture.Keys, fixture.KeysOther)
			testkit.NoError(tb, err, "a limited lookup succeeds")
			testkit.Len(tb, got, 1, "and returns no more than the limit")
		}),
		// `batchreader` is a structural stamp — variadic in, slice out — and the
		// count check reads it as the semantic claim that a batch read answers
		// once per key. This subject is a finder: it returns what it matched,
		// which is fewer results than keys and no error.
		//
		// Dropped rather than reshaped, because the subject is not wrong. It is
		// the same trade the miss family makes for `(T, bool)`: a shape common
		// enough to be stamped is not always the thing the stamp's check is
		// about, and a run that cannot tell them apart should say so by name.
		variadictest.FinderWithout(
			"Find/answers once per key",
			"FindWithLimit/answers once per key",
		),
	)
}

// Declining the double is separate from dropping a check, and a consumer who
// does not use the double should not pay for a second pass over every check.
//
// Run against the same subject because what is under test here is the harness
// rather than the implementation.
func TestFinderContractWithoutTheDouble(t *testing.T) {
	t.Parallel()

	variadictest.AssertFinderContract(t,
		variadictest.FinderSubject("in-memory", func() variadic.Finder {
			return variadictest.NewInMemory()
		}),
		variadictest.FinderWithout(
			"Find/an error carries the zero value",
			"FindWithLimit/an error carries the zero value",
			// The same structural stamp and the same finder, so the same two
			// drops: a run that declines the double still runs every check it
			// did not drop.
			"Find/answers once per key",
			"FindWithLimit/answers once per key",
		),
		variadictest.FinderWithoutDouble(),
	)
}
