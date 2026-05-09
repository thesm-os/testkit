// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package suite

import (
	"sync"
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/bindings"
)

// PureContext provides a typed factory and call function to Pure-shape
// primitives. A Pure-shaped method has no context parameter and no error
// return — func(...) T.
type PureContext[T any, R any] struct {
	T *testing.T
	bindings.PureBindings[T, R]
}

// PureAssertion is a typed conformance primitive for Pure-shaped methods.
type PureAssertion[T any, R any] func(PureContext[T, R])

// AssertPureReturns calls the pure method and asserts it returns the
// expected value.
func AssertPureReturns[T any, R comparable](want R) PureAssertion[T, R] {
	return func(ctx PureContext[T, R]) {
		ctx.T.Run("returns expected", func(t *testing.T) {
			t.Parallel()
			impl := ctx.Factory()
			got := ctx.Call(impl)
			testkit.Equal(t, got, want, "pure method must return expected value")
		})
	}
}

// AssertDeterministic calls the pure method N times on the same impl
// and asserts all results are equal. N must be >= 2.
func AssertDeterministic[T any, R comparable](n int) PureAssertion[T, R] {
	return func(ctx PureContext[T, R]) {
		ctx.T.Run("deterministic", func(t *testing.T) {
			if n < 2 {
				t.Fatalf("AssertDeterministic: n must be >= 2, got %d", n)
			}
			t.Parallel()
			impl := ctx.Factory()
			first := ctx.Call(impl)
			for i := 1; i < n; i++ {
				got := ctx.Call(impl)
				testkit.Equal(t, got, first, "pure method must be deterministic")
			}
		})
	}
}

// AssertNoSideEffects calls observe before and after the pure method,
// and asserts observable state did not change.
func AssertNoSideEffects[T, R any, S comparable](observe func(T) S) PureAssertion[T, R] {
	return func(ctx PureContext[T, R]) {
		ctx.T.Run("no side effects", func(t *testing.T) {
			t.Parallel()
			impl := ctx.Factory()
			before := observe(impl)
			_ = ctx.Call(impl)
			after := observe(impl)
			testkit.Equal(t, before, after, "pure method must not have side effects")
		})
	}
}

// AssertPureRespectsContext is the structural ctx-respect guarantee for
// Pure-shape methods. Pure has no context parameter, so cancellation is
// satisfied by signature: the impl cannot block on or observe a cancelled
// context. The subtest documents this and runs a smoke call to ensure
// the no-ctx contract isn't subverted by goroutine-local state.
func AssertPureRespectsContext[T, R any]() PureAssertion[T, R] {
	return func(ctx PureContext[T, R]) {
		ctx.T.Run("respects context (structural)", func(t *testing.T) {
			t.Parallel()
			impl := ctx.Factory()
			_ = ctx.Call(impl)
		})
	}
}

// AssertPureRejectInvalid is the structural reject-invalid guarantee for
// Pure-shape methods. Pure has no input parameter, so "invalid input" is
// not expressible — the function is total over its (empty) domain. The
// subtest documents this and runs a smoke call to assert no panic in
// what is the most degenerate input space.
func AssertPureRejectInvalid[T, R any]() PureAssertion[T, R] {
	return func(ctx PureContext[T, R]) {
		ctx.T.Run("rejects invalid (structural — no inputs)", func(t *testing.T) {
			t.Parallel()
			impl := ctx.Factory()
			_ = ctx.Call(impl)
		})
	}
}

// AssertPureConcurrentSafe runs the pure method from N goroutines
// concurrently. Pure methods should be intrinsically race-free; this
// primitive trips the race detector if an impl shares mutable state
// behind a misleadingly-pure signature.
func AssertPureConcurrentSafe[T, R any](workers, iters int) PureAssertion[T, R] {
	return func(ctx PureContext[T, R]) {
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

// AssertPureSmoke calls the Pure method once on a fresh impl. The
// subtest fails fast on panic, so a broken Factory or a method that
// panics on bare invocation surfaces as one localized failure before
// any contract assertion runs.
func AssertPureSmoke[T, R any]() PureAssertion[T, R] {
	return func(ctx PureContext[T, R]) {
		ctx.T.Run("smoke", func(t *testing.T) {
			t.Parallel()
			impl := ctx.Factory()
			_ = ctx.Call(impl)
		})
	}
}

// AssertPureBaseline runs the Pure-shape baseline: smoke, Returns(want),
// RespectsContext (structural), Deterministic over 3 calls, RejectInvalid
// (structural — no inputs), and ConcurrentSafe (4×10). Optional extras
// run after the deterministic check and before concurrency, so failures
// localize before fanout.
func AssertPureBaseline[T any, R comparable](
	want R,
	extra ...PureAssertion[T, R],
) PureAssertion[T, R] {
	return func(ctx PureContext[T, R]) {
		AssertPureSmoke[T, R]()(ctx)
		AssertPureReturns[T, R](want)(ctx)
		AssertPureRespectsContext[T, R]()(ctx)
		AssertDeterministic[T, R](3)(ctx)
		AssertPureRejectInvalid[T, R]()(ctx)
		for _, e := range extra {
			e(ctx)
		}
		AssertPureConcurrentSafe[T, R](4, 10)(ctx)
	}
}
