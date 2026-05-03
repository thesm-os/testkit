// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package testkit

import (
	"context"
	"errors"
	"testing"
)

// LifecycleBindings holds the reusable shape wiring for a Lifecycle-shaped method.
// Shared by suite (via LifecycleContext) and future generators (bench, model).
type LifecycleBindings[T any] struct {
	Factory func() T
	Call    func(context.Context, T) error
}

// LifecycleContext provides a typed factory and call function to
// Lifecycle-shape primitives. A Lifecycle-shaped method has the
// signature func(ctx) error.
type LifecycleContext[T any] struct {
	T *testing.T
	LifecycleBindings[T]
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
			NoError(t, err, "lifecycle call must succeed")
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
			AssertCtxCancellation(t, func(cancelledCtx context.Context) error {
				return ctx.Call(cancelledCtx, impl)
			})
		})
	}
}
