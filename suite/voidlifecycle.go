// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package suite

import (
	"context"
	"sync"
	"testing"

	"go.thesmos.sh/testkit/bindings"
)

// VoidLifecycleContext provides a typed factory and call function to
// VoidLifecycle-shape primitives. A VoidLifecycle-shaped method has the
// signature `func()` or `func(ctx)` — Reset, Close-without-error, and
// other parameterless lifecycle hooks.
type VoidLifecycleContext[T any] struct {
	T *testing.T
	bindings.VoidLifecycleBindings[T]
}

// VoidLifecycleAssertion is a typed conformance primitive for
// VoidLifecycle-shaped methods.
type VoidLifecycleAssertion[T any] func(VoidLifecycleContext[T])

// AssertVoidLifecycleSucceeds calls the method and asserts it does not
// panic. VoidLifecycle has no return position; the only observable
// happy-path contract is "no panic, no observable side-effect on this
// call alone."
func AssertVoidLifecycleSucceeds[T any]() VoidLifecycleAssertion[T] {
	return func(ctx VoidLifecycleContext[T]) {
		ctx.T.Run("succeeds", func(t *testing.T) {
			t.Parallel()
			impl := ctx.Factory()
			ctx.Call(t.Context(), impl)
		})
	}
}

// AssertVoidLifecycleIdempotent calls the method twice and asserts no
// panic.
func AssertVoidLifecycleIdempotent[T any]() VoidLifecycleAssertion[T] {
	return func(ctx VoidLifecycleContext[T]) {
		ctx.T.Run("idempotent", func(t *testing.T) {
			t.Parallel()
			impl := ctx.Factory()
			ctx.Call(t.Context(), impl)
			ctx.Call(t.Context(), impl)
		})
	}
}

// AssertVoidLifecycleRespectsContext is the structural ctx-respect
// guarantee. VoidLifecycle has no return position, so cancellation
// cannot be surfaced as an error — the contract is "call under cancel
// must not panic and must not block."
func AssertVoidLifecycleRespectsContext[T any]() VoidLifecycleAssertion[T] {
	return func(ctx VoidLifecycleContext[T]) {
		ctx.T.Run("respects context", func(t *testing.T) {
			t.Parallel()
			impl := ctx.Factory()
			cctx, cancel := context.WithCancel(t.Context())
			cancel()
			ctx.Call(cctx, impl)
		})
	}
}

// AssertVoidLifecycleRejectInvalid runs the method against a known-
// invalid factory and runs a consumer-supplied check that observes the
// rejection. VoidLifecycle has no return position; the consumer's check
// observes via a paired reader/aggregator.
func AssertVoidLifecycleRejectInvalid[T any](
	invalidFactory func() T,
	check func(t *testing.T, impl T),
) VoidLifecycleAssertion[T] {
	return func(ctx VoidLifecycleContext[T]) {
		ctx.T.Run("rejects invalid", func(t *testing.T) {
			t.Parallel()
			impl := invalidFactory()
			ctx.Call(t.Context(), impl)
			check(t, impl)
		})
	}
}

// AssertVoidLifecycleRejectInvalidWith runs the method against an
// invalid-factory impl and asserts no panic. VoidLifecycle has no
// return position; the contract under "invalid" input is therefore
// the no-panic guarantee — observable rejection (the mutation didn't
// take effect) belongs to the cross-method invariant directives that
// pair the writer with a reader.
func AssertVoidLifecycleRejectInvalidWith[T any](invalidFactory func() T) VoidLifecycleAssertion[T] {
	return func(ctx VoidLifecycleContext[T]) {
		ctx.T.Run("rejects invalid", func(t *testing.T) {
			t.Parallel()
			impl := invalidFactory()
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("VoidLifecycle on invalid impl must not panic, got: %v", r)
				}
			}()
			ctx.Call(t.Context(), impl)
		})
	}
}

// AssertVoidLifecycleConcurrentSafe runs the method from N goroutines
// concurrently.
func AssertVoidLifecycleConcurrentSafe[T any](workers, iters int) VoidLifecycleAssertion[T] {
	return func(ctx VoidLifecycleContext[T]) {
		ctx.T.Run("concurrent safe", func(t *testing.T) {
			t.Parallel()
			impl := ctx.Factory()
			var wg sync.WaitGroup
			for range workers {
				wg.Go(func() {
					for range iters {
						ctx.Call(t.Context(), impl)
					}
				})
			}
			wg.Wait()
		})
	}
}

// AssertVoidLifecycleSmoke calls the method once on a fresh impl. The
// subtest fails fast on panic, surfacing a broken Factory or a method
// that panics on bare invocation as one localized failure before any
// contract assertion runs.
func AssertVoidLifecycleSmoke[T any]() VoidLifecycleAssertion[T] {
	return func(ctx VoidLifecycleContext[T]) {
		ctx.T.Run("smoke", func(t *testing.T) {
			t.Parallel()
			impl := ctx.Factory()
			ctx.Call(t.Context(), impl)
		})
	}
}

// AssertVoidLifecycleBaseline runs the VoidLifecycle-shape baseline:
// smoke, Succeeds, RespectsContext, Idempotent, and ConcurrentSafe
// (4×10). Optional extras (e.g. RejectInvalidWith under an
// InvalidFactory) run between idempotency and concurrency.
func AssertVoidLifecycleBaseline[T any](
	extra ...VoidLifecycleAssertion[T],
) VoidLifecycleAssertion[T] {
	return func(ctx VoidLifecycleContext[T]) {
		AssertVoidLifecycleSmoke[T]()(ctx)
		AssertVoidLifecycleSucceeds[T]()(ctx)
		AssertVoidLifecycleRespectsContext[T]()(ctx)
		AssertVoidLifecycleIdempotent[T]()(ctx)
		for _, e := range extra {
			e(ctx)
		}
		AssertVoidLifecycleConcurrentSafe[T](4, 10)(ctx)
	}
}
