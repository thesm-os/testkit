// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package testkit

import (
	"testing"
)

// PredicateContext provides a typed factory and call function to
// Predicate-shape primitives. A Predicate-shaped method returns bool only.
type PredicateContext[T any] struct {
	T       *testing.T
	Factory func() T
	Call    func(T) bool
}

// PredicateAssertion is a typed conformance primitive for Predicate-shaped methods.
type PredicateAssertion[T any] func(PredicateContext[T])

// AssertPredicateConsistent calls the predicate N times on the same impl
// and asserts all results are equal. N must be >= 2.
func AssertPredicateConsistent[T any](n int) PredicateAssertion[T] {
	return func(ctx PredicateContext[T]) {
		ctx.T.Run("predicate consistent", func(t *testing.T) {
			if n < 2 {
				t.Fatalf("AssertPredicateConsistent: n must be >= 2, got %d", n)
			}
			t.Parallel()
			impl := ctx.Factory()
			first := ctx.Call(impl)
			for i := 1; i < n; i++ {
				got := ctx.Call(impl)
				Equal(t, got, first, "predicate must be consistent")
			}
		})
	}
}

// AssertPredicateReturns calls the predicate and asserts it returns
// the expected value.
func AssertPredicateReturns[T any](want bool) PredicateAssertion[T] {
	return func(ctx PredicateContext[T]) {
		ctx.T.Run("predicate returns expected", func(t *testing.T) {
			t.Parallel()
			impl := ctx.Factory()
			got := ctx.Call(impl)
			Equal(t, got, want, "predicate must return expected value")
		})
	}
}
