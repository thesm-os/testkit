// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package suite

import (
	"context"
	"sync"
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/bindings"
)

// MultiArgWriterContext provides a typed factory and call function to
// MultiArgWriter-shape primitives at exactly 3 non-ctx params:
// `func(ctx, P1, P2, P3) error`. Use this when you want type safety
// at the call site — typically when writing the assertion by hand.
//
// For arity-agnostic generator-emitted contracts, use
// [MultiArgWriterVariadicContext] which accepts any non-ctx arity.
type MultiArgWriterContext[T any, P1, P2, P3 any] struct {
	T *testing.T
	bindings.MultiArgWriterBindings[T, P1, P2, P3]
}

// MultiArgWriterAssertion is a typed conformance primitive for
// MultiArgWriter-shape methods at 3 non-ctx params.
type MultiArgWriterAssertion[T any, P1, P2, P3 any] func(MultiArgWriterContext[T, P1, P2, P3])

// AssertMultiArgWriteSucceeds writes the given args and asserts no error.
func AssertMultiArgWriteSucceeds[T, P1, P2, P3 any](
	p1 P1,
	p2 P2,
	p3 P3,
) MultiArgWriterAssertion[T, P1, P2, P3] {
	return func(ctx MultiArgWriterContext[T, P1, P2, P3]) {
		ctx.T.Run("write succeeds", func(t *testing.T) {
			t.Parallel()
			impl := ctx.Factory()
			err := ctx.Call(t.Context(), impl, p1, p2, p3)
			testkit.NoError(t, err, "multi-arg write must succeed")
		})
	}
}

// AssertMultiArgWriteRejectInvalid writes invalid args and asserts the
// configured sentinel is returned.
func AssertMultiArgWriteRejectInvalid[T, P1, P2, P3 any](
	p1 P1,
	p2 P2,
	p3 P3,
	sentinel error,
) MultiArgWriterAssertion[T, P1, P2, P3] {
	return func(ctx MultiArgWriterContext[T, P1, P2, P3]) {
		ctx.T.Run("rejects invalid", func(t *testing.T) {
			t.Parallel()
			impl := ctx.Factory()
			err := ctx.Call(t.Context(), impl, p1, p2, p3)
			testkit.ErrorIs(t, err, sentinel,
				"multi-arg writer must surface sentinel for invalid args")
		})
	}
}

// AssertMultiArgWriterIdempotent writes the same args twice and asserts
// both calls succeed.
func AssertMultiArgWriterIdempotent[T, P1, P2, P3 any](
	p1 P1,
	p2 P2,
	p3 P3,
) MultiArgWriterAssertion[T, P1, P2, P3] {
	return func(ctx MultiArgWriterContext[T, P1, P2, P3]) {
		ctx.T.Run("idempotent", func(t *testing.T) {
			t.Parallel()
			impl := ctx.Factory()
			err := ctx.Call(t.Context(), impl, p1, p2, p3)
			testkit.NoError(t, err, "first write must succeed")
			err = ctx.Call(t.Context(), impl, p1, p2, p3)
			testkit.NoError(t, err, "second write of same args must succeed (idempotent)")
		})
	}
}

// AssertMultiArgWriterRespectsContext invokes the writer with a cancelled
// context and asserts context.Canceled is returned.
func AssertMultiArgWriterRespectsContext[T, P1, P2, P3 any](
	p1 P1,
	p2 P2,
	p3 P3,
) MultiArgWriterAssertion[T, P1, P2, P3] {
	return func(ctx MultiArgWriterContext[T, P1, P2, P3]) {
		ctx.T.Run("respects context", func(t *testing.T) {
			t.Parallel()
			impl := ctx.Factory()
			cctx, cancel := context.WithCancel(t.Context())
			cancel()
			err := ctx.Call(cctx, impl, p1, p2, p3)
			testkit.ErrorIs(t, err, context.Canceled,
				"multi-arg writer must surface ctx.Canceled when called with a cancelled context")
		})
	}
}

// AssertMultiArgWriterConcurrentSafe runs the writer from N goroutines
// concurrently.
func AssertMultiArgWriterConcurrentSafe[T, P1, P2, P3 any](
	p1 P1,
	p2 P2,
	p3 P3,
	workers, iters int,
) MultiArgWriterAssertion[T, P1, P2, P3] {
	return func(ctx MultiArgWriterContext[T, P1, P2, P3]) {
		ctx.T.Run("concurrent safe", func(t *testing.T) {
			t.Parallel()
			impl := ctx.Factory()
			var wg sync.WaitGroup
			for range workers {
				wg.Go(func() {
					for range iters {
						_ = ctx.Call(t.Context(), impl, p1, p2, p3)
					}
				})
			}
			wg.Wait()
		})
	}
}

// MultiArgWriterVariadicContext provides arity-agnostic factory + call
// wiring for a MultiArgWriter-shape method. The underlying bindings
// store the call closure as `func(ctx, T, ...any) error` — the
// generator emits a per-method typed wrapper that converts `[]any`
// back to typed args at the call site, so the consumer-facing
// contract retains type safety while the runtime exercises any
// non-ctx arity from one set of assertions.
//
// Bench keeps the typed [MultiArgWriterContext] above —
// boxing through `any` defeats hot-path measurements.
type MultiArgWriterVariadicContext[T any] struct {
	T *testing.T
	bindings.MultiArgWriterVariadicBindings[T]
}

// MultiArgWriterVariadicAssertion is a typed conformance primitive
// for MultiArgWriter-shape methods at any non-ctx arity. Generators
// emit per-method typed wrappers that convert `[]any` back to typed
// args at the call site, restoring type safety at the consumer
// boundary.
type MultiArgWriterVariadicAssertion[T any] func(MultiArgWriterVariadicContext[T])

// AssertMultiArgWriteSucceedsVariadic writes the given args and
// asserts no error. Variadic counterpart of [AssertMultiArgWriteSucceeds].
func AssertMultiArgWriteSucceedsVariadic[T any](args ...any) MultiArgWriterVariadicAssertion[T] {
	return func(ctx MultiArgWriterVariadicContext[T]) {
		ctx.T.Run("write succeeds", func(t *testing.T) {
			t.Parallel()
			impl := ctx.Factory()
			err := ctx.Call(t.Context(), impl, args...)
			testkit.NoError(t, err, "multi-arg write must succeed")
		})
	}
}

// AssertMultiArgWriteRejectInvalidVariadic writes invalid args and
// asserts the configured sentinel is returned. Variadic counterpart
// of [AssertMultiArgWriteRejectInvalid].
func AssertMultiArgWriteRejectInvalidVariadic[T any](
	sentinel error,
	args ...any,
) MultiArgWriterVariadicAssertion[T] {
	return func(ctx MultiArgWriterVariadicContext[T]) {
		ctx.T.Run("rejects invalid", func(t *testing.T) {
			t.Parallel()
			impl := ctx.Factory()
			err := ctx.Call(t.Context(), impl, args...)
			testkit.ErrorIs(t, err, sentinel,
				"multi-arg writer must surface sentinel for invalid args")
		})
	}
}

// AssertMultiArgWriterIdempotentVariadic writes the same args twice and
// asserts both calls succeed. Variadic counterpart of
// [AssertMultiArgWriterIdempotent].
func AssertMultiArgWriterIdempotentVariadic[T any](args ...any) MultiArgWriterVariadicAssertion[T] {
	return func(ctx MultiArgWriterVariadicContext[T]) {
		ctx.T.Run("idempotent", func(t *testing.T) {
			t.Parallel()
			impl := ctx.Factory()
			err := ctx.Call(t.Context(), impl, args...)
			testkit.NoError(t, err, "first write must succeed")
			err = ctx.Call(t.Context(), impl, args...)
			testkit.NoError(t, err, "second write of same args must succeed (idempotent)")
		})
	}
}

// AssertMultiArgWriterRespectsContextVariadic invokes the writer with
// a cancelled context and asserts context.Canceled is returned.
// Variadic counterpart of [AssertMultiArgWriterRespectsContext].
func AssertMultiArgWriterRespectsContextVariadic[T any](args ...any) MultiArgWriterVariadicAssertion[T] {
	return func(ctx MultiArgWriterVariadicContext[T]) {
		ctx.T.Run("respects context", func(t *testing.T) {
			t.Parallel()
			impl := ctx.Factory()
			cctx, cancel := context.WithCancel(t.Context())
			cancel()
			err := ctx.Call(cctx, impl, args...)
			testkit.ErrorIs(t, err, context.Canceled,
				"multi-arg writer must surface ctx.Canceled when called with a cancelled context")
		})
	}
}

// AssertMultiArgWriterConcurrentSafeVariadic runs the writer from N
// goroutines concurrently. Variadic counterpart of
// [AssertMultiArgWriterConcurrentSafe].
func AssertMultiArgWriterConcurrentSafeVariadic[T any](
	workers, iters int,
	args ...any,
) MultiArgWriterVariadicAssertion[T] {
	return func(ctx MultiArgWriterVariadicContext[T]) {
		ctx.T.Run("concurrent safe", func(t *testing.T) {
			t.Parallel()
			impl := ctx.Factory()
			var wg sync.WaitGroup
			for range workers {
				wg.Go(func() {
					for range iters {
						_ = ctx.Call(t.Context(), impl, args...)
					}
				})
			}
			wg.Wait()
		})
	}
}
