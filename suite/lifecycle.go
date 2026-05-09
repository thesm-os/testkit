// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package suite

import (
	"context"
	"errors"
	"sync"
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/bindings"
)

// LifecycleContext provides a typed factory and call function to
// Lifecycle-shape primitives. A Lifecycle-shaped method has the
// signature func(ctx) error.
type LifecycleContext[T any] struct {
	T *testing.T
	bindings.LifecycleBindings[T]
}

// LifecycleAssertion is a typed conformance primitive for Lifecycle-shaped methods.
type LifecycleAssertion[T any] func(LifecycleContext[T])

// AssertLifecycleSucceeds calls the lifecycle method and asserts no error.
func AssertLifecycleSucceeds[T any]() LifecycleAssertion[T] {
	return func(ctx LifecycleContext[T]) {
		ctx.T.Run("lifecycle succeeds", func(t *testing.T) {
			t.Parallel()
			impl := ctx.Factory()
			err := ctx.Call(t.Context(), impl)
			testkit.NoError(t, err, "lifecycle call must succeed")
		})
	}
}

// AssertLifecycleIdempotent calls the lifecycle method twice and asserts
// both calls succeed (or both return the same error).
func AssertLifecycleIdempotent[T any]() LifecycleAssertion[T] {
	return func(ctx LifecycleContext[T]) {
		ctx.T.Run("lifecycle idempotent", func(t *testing.T) {
			t.Parallel()
			impl := ctx.Factory()
			err1 := ctx.Call(t.Context(), impl)
			err2 := ctx.Call(t.Context(), impl)
			if err1 == nil && err2 == nil {
				return
			}
			if err1 != nil && err2 != nil && (errors.Is(err1, err2) || errors.Is(err2, err1)) {
				return
			}
			t.Errorf("lifecycle must be idempotent: first=%v, second=%v", err1, err2)
		})
	}
}

// AssertLifecycleRespectsContext calls the lifecycle method with a cancelled
// context and asserts it returns a context error.
func AssertLifecycleRespectsContext[T any]() LifecycleAssertion[T] {
	return func(ctx LifecycleContext[T]) {
		ctx.T.Run("lifecycle respects context", func(t *testing.T) {
			t.Parallel()
			impl := ctx.Factory()
			testkit.AssertCtxCancellation(t, func(cancelledCtx context.Context) error {
				return ctx.Call(cancelledCtx, impl)
			})
		})
	}
}

// AssertLifecycleRejectInvalid calls the lifecycle method against a factory
// that produces an invalid-state impl and asserts the call returns the
// configured sentinel. The consumer supplies an invalidFactory that yields
// an impl in a known-bad state (e.g. an already-closed connection); the
// contract is that the lifecycle call refuses to operate and surfaces the
// sentinel rather than silently no-op'ing.
func AssertLifecycleRejectInvalid[T any](invalidFactory func() T, sentinel error) LifecycleAssertion[T] {
	return func(ctx LifecycleContext[T]) {
		ctx.T.Run("lifecycle rejects invalid", func(t *testing.T) {
			t.Parallel()
			impl := invalidFactory()
			err := ctx.Call(t.Context(), impl)
			testkit.ErrorIs(t, err, sentinel,
				"lifecycle must surface the configured sentinel against an invalid-state impl")
		})
	}
}

// AssertLifecycleRejectInvalidWith calls the lifecycle method against an
// invalid-factory impl and asserts the call returns a non-nil error.
// Use this when the method has no `//testkit:errors` sentinel — the
// contract is just "errors on invalid input" without naming a specific
// sentinel. When a sentinel is declared, prefer
// [AssertLifecycleRejectInvalid] which asserts errors.Is against it.
func AssertLifecycleRejectInvalidWith[T any](invalidFactory func() T) LifecycleAssertion[T] {
	return func(ctx LifecycleContext[T]) {
		ctx.T.Run("lifecycle rejects invalid", func(t *testing.T) {
			t.Parallel()
			impl := invalidFactory()
			err := ctx.Call(t.Context(), impl)
			testkit.True(t, err != nil,
				"lifecycle on invalid impl must return a non-nil error")
		})
	}
}

// AssertLifecycleConcurrentSafe runs the lifecycle method from N goroutines
// concurrently.
func AssertLifecycleConcurrentSafe[T any](workers, iters int) LifecycleAssertion[T] {
	return func(ctx LifecycleContext[T]) {
		ctx.T.Run("concurrent safe", func(t *testing.T) {
			t.Parallel()
			impl := ctx.Factory()
			var wg sync.WaitGroup
			for range workers {
				wg.Go(func() {
					for range iters {
						_ = ctx.Call(t.Context(), impl)
					}
				})
			}
			wg.Wait()
		})
	}
}

// AssertLifecycleSmoke calls the lifecycle method once on a fresh
// impl. The subtest fails fast on panic, surfacing a broken Factory or
// a method that panics on bare invocation as one localized failure
// before any contract assertion runs.
func AssertLifecycleSmoke[T any]() LifecycleAssertion[T] {
	return func(ctx LifecycleContext[T]) {
		ctx.T.Run("smoke", func(t *testing.T) {
			t.Parallel()
			impl := ctx.Factory()
			_ = ctx.Call(t.Context(), impl)
		})
	}
}

// AssertLifecycleBaseline runs the Lifecycle-shape baseline: smoke,
// Succeeds, RespectsContext, Idempotent, and ConcurrentSafe (4×10).
// Optional extras (e.g. RejectInvalid or RejectInvalidWith under an
// InvalidFactory) run between idempotency and concurrency.
func AssertLifecycleBaseline[T any](
	extra ...LifecycleAssertion[T],
) LifecycleAssertion[T] {
	return func(ctx LifecycleContext[T]) {
		AssertLifecycleSmoke[T]()(ctx)
		AssertLifecycleSucceeds[T]()(ctx)
		AssertLifecycleRespectsContext[T]()(ctx)
		AssertLifecycleIdempotent[T]()(ctx)
		for _, e := range extra {
			e(ctx)
		}
		AssertLifecycleConcurrentSafe[T](4, 10)(ctx)
	}
}
