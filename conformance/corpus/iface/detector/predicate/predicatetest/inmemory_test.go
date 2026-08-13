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
		predicatetest.PredicateModel(),
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

// The saturation prover: every bound law must be able to fail as itself,
// a defect worn on its own methods reddening the run by name.
func TestPredicateSaturation(t *testing.T) {
	t.Parallel()
	predicatetest.PredicateModelSaturation(t, func() predicate.Predicate {
		return predicatetest.NewInMemory()
	})
}
