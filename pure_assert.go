// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package testkit

import (
	"testing"
)

// PureBindings holds the reusable shape wiring for a Pure-shaped method.
// Shared by suite (via PureContext) and future generators (bench, model).
type PureBindings[T any, R any] struct {
	Factory func() T
	Call    func(T) R
}

// PureContext provides a typed factory and call function to Pure-shape
// primitives. A Pure-shaped method has no context parameter and no error
// return — func(...) T.
type PureContext[T any, R any] struct {
	T *testing.T
	PureBindings[T, R]
}

// PureAssertion is a typed conformance primitive for Pure-shaped methods.
type PureAssertion[T any, R any] func(PureContext[T, R])

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
				Equal(t, got, first, "pure method must be deterministic")
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
			Equal(t, before, after, "pure method must not have side effects")
		})
	}
}
