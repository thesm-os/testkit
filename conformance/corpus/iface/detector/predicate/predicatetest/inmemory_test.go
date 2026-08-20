// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package predicatetest_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/conformance/corpus/iface/detector/predicate"
	"go.thesmos.sh/testkit/conformance/corpus/iface/detector/predicate/predicatetest"
)

// A bare predicate earns one derived check: it takes nothing to vary and
// reports nothing to compare, so the smoke call is the whole signature
// family. Which way it should answer is the row's.
func TestPredicateContract(t *testing.T) {
	t.Parallel()

	predicatetest.RunPredicate(t,
		predicatetest.PredicateHarness[*predicatetest.InMemory]{Name: "in-memory", New: predicatetest.NewInMemory},
		predicatetest.PredicateChecks{
			{
				Method: "IsEmpty",
				Name:   "true-on-a-fresh-subject",
				Claim:  "IsEmpty reports true for a fresh subject",
				Run: func(tb testing.TB, s predicate.Predicate, fx predicatetest.PredicateFixture) {
					tb.Helper()
					testkit.True(tb, s.IsEmpty(), "nothing has been added yet")
				},
			},
		},
	)
}

// Dropping a check is written against the typed index rather than a string, so
// a check that is renamed or stops being emitted breaks this compile instead of
// silently declining nothing.
func TestPredicateContractWithoutSmoke(t *testing.T) {
	t.Parallel()

	predicatetest.RunPredicate(t,
		predicatetest.PredicateHarness[*predicatetest.InMemory]{Name: "in-memory", New: predicatetest.NewInMemory},
		predicatetest.PredicateSuite.Without(predicatetest.PredicateSuite.Checks.IsEmpty.Smoke()),
	)
}
