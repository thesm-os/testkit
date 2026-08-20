// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package variadictest_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/conformance/corpus/iface/lang/variadic"
	"go.thesmos.sh/testkit/conformance/corpus/iface/lang/variadic/variadictest"
)

// A variadic parameter is one fixture field, so a generated check witnesses one
// element. Everything about *several* is the author's to state, which is what
// the rows below are.
func TestFinderContract(t *testing.T) {
	t.Parallel()

	fx := variadictest.DefaultFinderFixture()

	variadictest.RunFinder(t,
		variadictest.FinderHarness[*variadictest.InMemory]{
			Name: "in-memory",
			New: func() *variadictest.InMemory {
				s := variadictest.NewInMemory()
				s.Put(fx.Keys(), "first")
				s.Put(fx.KeysOther(), "second")
				return s
			},
		},
		variadictest.FinderChecks{
			{
				Method: "Find",
				Name:   "one-result-per-held-key",
				Claim:  "Find returns one result per key it holds, in order",
				Run: func(tb testing.TB, s variadic.Finder, fx variadictest.FinderFixture) {
					tb.Helper()
					got, err := s.Find(tb.Context(), fx.Keys(), fx.KeysOther())
					testkit.NoError(tb, err, "a lookup of two held keys succeeds")
					testkit.Equal(tb, got, []string{"first", "second"},
						"several keys are answered in the order asked")
				},
			},
			{
				Method: "Find",
				Name:   "empty-lookup-is-reported",
				Claim:  "Find reports a lookup with nothing to look up",
				Run: func(tb testing.TB, s variadic.Finder, fx variadictest.FinderFixture) {
					tb.Helper()
					_, err := s.Find(tb.Context())
					testkit.ErrorIs(tb, err, variadictest.ErrNoKeys,
						"the empty variadic form is the one a derived check cannot reach")
				},
			},
			{
				Method: "FindWithLimit",
				Name:   "truncates-to-the-limit",
				Claim:  "FindWithLimit truncates to the limit",
				Run: func(tb testing.TB, s variadic.Finder, fx variadictest.FinderFixture) {
					tb.Helper()
					got, err := s.FindWithLimit(tb.Context(), 1, fx.Keys(), fx.KeysOther())
					testkit.NoError(tb, err, "a limited lookup succeeds")
					testkit.Len(tb, got, 1, "and returns no more than the limit")
				},
			},
		},
	)
}

// Dropping a check is written against the typed index rather than a string, so
// a check that is renamed or stops being emitted breaks this compile instead of
// silently declining nothing.
func TestFinderContractWithoutSmoke(t *testing.T) {
	t.Parallel()

	variadictest.RunFinder(t,
		variadictest.FinderHarness[*variadictest.InMemory]{Name: "in-memory", New: variadictest.NewInMemory},
		variadictest.FinderSuite.Without(variadictest.FinderSuite.Checks.Find.Smoke()),
	)
}
