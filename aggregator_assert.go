// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package testkit

import (
	"cmp"
	"context"
	"testing"
)

// AggregatorBindings holds the reusable shape wiring for an Aggregator-shaped method.
// Shared substrate for shape-driven generators.
type AggregatorBindings[T any, R any] struct {
	Factory func() T
	Call    func(context.Context, T) (R, error)
}

// AggregatorContext provides a typed factory and call function to
// Aggregator-shape primitives. An Aggregator-shaped method has the
// signature func(ctx) (T, error) — no key, scalar return.
type AggregatorContext[T any, R any] struct {
	T *testing.T
	AggregatorBindings[T, R]
}

// AggregatorAssertion is a typed conformance primitive for Aggregator-shaped methods.
type AggregatorAssertion[T any, R any] func(AggregatorContext[T, R])

// AssertAggregatorReturns calls the aggregator and asserts it returns the
// expected value with no error.
func AssertAggregatorReturns[T any, R comparable](want R) AggregatorAssertion[T, R] {
	return func(ctx AggregatorContext[T, R]) {
		ctx.T.Run("aggregator returns expected", func(t *testing.T) {
			t.Parallel()
			impl := ctx.Factory()
			got, err := ctx.Call(t.Context(), impl)
			NoError(t, err, "aggregator must not error")
			Equal(t, got, want, "aggregator must return expected value")
		})
	}
}

// AssertAggregatorBounded calls the aggregator and asserts the result
// is in [lower, upper].
func AssertAggregatorBounded[T any, R cmp.Ordered](lower, upper R) AggregatorAssertion[T, R] {
	return func(ctx AggregatorContext[T, R]) {
		ctx.T.Run("aggregator bounded", func(t *testing.T) {
			t.Parallel()
			impl := ctx.Factory()
			got, err := ctx.Call(t.Context(), impl)
			NoError(t, err, "aggregator must not error")
			AssertBounded(t, lower, upper, func() R { return got })
		})
	}
}

// AssertAggregatorConsistent calls the aggregator N times and asserts
// all results are equal. N must be >= 2.
func AssertAggregatorConsistent[T any, R comparable](n int) AggregatorAssertion[T, R] {
	return func(ctx AggregatorContext[T, R]) {
		ctx.T.Run("aggregator consistent", func(t *testing.T) {
			if n < 2 {
				t.Fatalf("AssertAggregatorConsistent: n must be >= 2, got %d", n)
			}
			t.Parallel()
			impl := ctx.Factory()
			first, err := ctx.Call(t.Context(), impl)
			NoError(t, err, "first call must not error")
			for i := 1; i < n; i++ {
				got, err := ctx.Call(t.Context(), impl)
				NoError(t, err, "call must not error")
				Equal(t, got, first, "aggregator must be consistent")
			}
		})
	}
}
