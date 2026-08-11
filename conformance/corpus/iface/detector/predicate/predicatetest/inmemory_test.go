// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package predicatetest_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/conformance/corpus/iface/detector/predicate"
	"go.thesmos.sh/testkit/conformance/corpus/iface/detector/predicate/predicatetest"
)

// The floor of what a harness can owe: no context, no error, no parameter, so
// one smoke call is the entire generated contract.
//
// Worth generating anyway. A method that panics on a fresh subject is one no
// other check in the file would reach, and the smoke call costs a line. What the
// predicate actually means — that the answer tracks the state — is a claim over
// two calls, so it is written here.
func TestPredicateContract(t *testing.T) {
	t.Parallel()

	predicatetest.AssertPredicateContract(t,
		predicatetest.PredicateSubject("in-memory", func() predicate.Predicate {
			return predicatetest.NewInMemory()
		}),
		predicatetest.PredicateOnIsEmpty("reports true for a fresh subject", func(
			tb testing.TB, subject predicate.Predicate,
		) {
			tb.Helper()
			testkit.True(tb, subject.IsEmpty(), "nothing has been added yet")
		}),
	)
}

// The answer has to move, or the predicate is a constant that passes every
// check asking only that it not panic.
func TestIsEmptyTracksTheState(t *testing.T) {
	t.Parallel()

	s := predicatetest.NewInMemory()
	testkit.True(t, s.IsEmpty(), "a fresh subject holds nothing")
	s.Add("item")
	testkit.False(t, s.IsEmpty(), "and reports so once it does")
}

// Declining the double is separate from dropping a check.
func TestPredicateContractWithoutTheDouble(t *testing.T) {
	t.Parallel()

	predicatetest.AssertPredicateContract(t,
		predicatetest.PredicateSubject("in-memory", func() predicate.Predicate {
			return predicatetest.NewInMemory()
		}),
		predicatetest.PredicateWithout("IsEmpty/smoke"),
		predicatetest.PredicateWithoutDouble(),
	)
}
