// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package suite

import (
	"sync"
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/bindings"
)

// PredicateContext provides a typed factory and call function to
// Predicate-shape primitives. A Predicate-shaped method returns bool only.
type PredicateContext[T any] struct {
	T *testing.T
	bindings.PredicateBindings[T]
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
				testkit.Equal(t, got, first, "predicate must be consistent")
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
			testkit.Equal(t, got, want, sampleAlignmentHint("Predicate", "the predicate"))
		})
	}
}

// AssertPredicateRespectsContext is the structural ctx-respect guarantee
// for Predicate-shape methods. Predicate has no context parameter, so
// cancellation is satisfied by signature; the subtest runs a smoke call
// to ensure no goroutine-local context state subverts the no-ctx contract.
func AssertPredicateRespectsContext[T any]() PredicateAssertion[T] {
	return func(ctx PredicateContext[T]) {
		ctx.T.Run("respects context (structural)", func(t *testing.T) {
			t.Parallel()
			impl := ctx.Factory()
			_ = ctx.Call(impl)
		})
	}
}

// AssertPredicateRejectInvalid is the structural reject-invalid guarantee
// for Predicate-shape methods. Predicate has no input — the function is
// total over its (empty) domain.
func AssertPredicateRejectInvalid[T any]() PredicateAssertion[T] {
	return func(ctx PredicateContext[T]) {
		ctx.T.Run("rejects invalid (structural — no inputs)", func(t *testing.T) {
			t.Parallel()
			impl := ctx.Factory()
			_ = ctx.Call(impl)
		})
	}
}

// AssertPredicateConcurrentSafe runs the predicate from N goroutines
// concurrently. Predicates should be race-free; this primitive trips
// the race detector if an impl shares mutable state behind a
// misleadingly-pure signature.
func AssertPredicateConcurrentSafe[T any](workers, iters int) PredicateAssertion[T] {
	return func(ctx PredicateContext[T]) {
		ctx.T.Run("concurrent safe", func(t *testing.T) {
			t.Parallel()
			impl := ctx.Factory()
			var wg sync.WaitGroup
			for range workers {
				wg.Go(func() {
					for range iters {
						_ = ctx.Call(impl)
					}
				})
			}
			wg.Wait()
		})
	}
}

// AssertPredicateSmoke calls the Predicate method once on a fresh
// impl. The subtest fails fast on panic, so a broken Factory or a
// method that panics on bare invocation surfaces as one localized
// failure before any contract assertion runs.
func AssertPredicateSmoke[T any]() PredicateAssertion[T] {
	return func(ctx PredicateContext[T]) {
		ctx.T.Run("smoke", func(t *testing.T) {
			t.Parallel()
			impl := ctx.Factory()
			_ = ctx.Call(impl)
		})
	}
}

// AssertPredicateBaseline runs the Predicate-shape baseline: smoke,
// Returns(want), RespectsContext (structural), Consistent over 3 calls,
// RejectInvalid (structural — no inputs), and ConcurrentSafe (4×10).
// Optional extras run between consistency and concurrency.
func AssertPredicateBaseline[T any](
	want bool,
	extra ...PredicateAssertion[T],
) PredicateAssertion[T] {
	return func(ctx PredicateContext[T]) {
		AssertPredicateSmoke[T]()(ctx)
		AssertPredicateReturns[T](want)(ctx)
		AssertPredicateRespectsContext[T]()(ctx)
		AssertPredicateConsistent[T](3)(ctx)
		AssertPredicateRejectInvalid[T]()(ctx)
		for _, e := range extra {
			e(ctx)
		}
		AssertPredicateConcurrentSafe[T](4, 10)(ctx)
	}
}
