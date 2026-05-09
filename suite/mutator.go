// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package suite

import (
	"context"
	"sync"
	"testing"

	"go.thesmos.sh/testkit/bindings"
)

// MutatorContext provides a typed factory and call function to
// Mutator-shape primitives. A Mutator-shaped method has the
// signature func(ctx, V) with no return value.
type MutatorContext[T, V any] struct {
	T *testing.T
	bindings.MutatorBindings[T, V]
}

// MutatorAssertion is a typed conformance primitive for Mutator-shaped methods.
type MutatorAssertion[T, V any] func(MutatorContext[T, V])

// AssertMutatorSucceeds calls the mutator with a sample value and
// asserts it does not panic.
func AssertMutatorSucceeds[T, V any](sample V) MutatorAssertion[T, V] {
	return func(ctx MutatorContext[T, V]) {
		ctx.T.Run("mutator succeeds", func(t *testing.T) {
			t.Parallel()
			impl := ctx.Factory()
			ctx.Call(t.Context(), impl, sample)
		})
	}
}

// AssertMutatorIdempotent calls the mutator twice with the same value
// and asserts neither call panics.
func AssertMutatorIdempotent[T, V any](sample V) MutatorAssertion[T, V] {
	return func(ctx MutatorContext[T, V]) {
		ctx.T.Run("mutator idempotent", func(t *testing.T) {
			t.Parallel()
			impl := ctx.Factory()
			ctx.Call(t.Context(), impl, sample)
			ctx.Call(t.Context(), impl, sample)
		})
	}
}

// AssertMutatorRespectsContext calls the mutator with an already-
// cancelled context and asserts the call does not panic. Mutators
// have no return position to surface ctx.Canceled — the contract
// under cancellation is "no observable side-effect"; impls that
// need to surface the cancel must use a different shape (e.g.
// Lifecycle returning error).
func AssertMutatorRespectsContext[T, V any](sample V) MutatorAssertion[T, V] {
	return func(ctx MutatorContext[T, V]) {
		ctx.T.Run("respects context", func(t *testing.T) {
			t.Parallel()
			impl := ctx.Factory()
			cctx, cancel := context.WithCancel(t.Context())
			cancel()
			ctx.Call(cctx, impl, sample)
		})
	}
}

// AssertMutatorRejectInvalid calls the mutator with an invalid value and
// runs a consumer-supplied check that observes the impl rejected the
// invalid input. The check is consumer-driven because Mutators have no
// return position — the rejection must be observed via a paired
// reader/aggregator/predicate.
func AssertMutatorRejectInvalid[T, V any](
	invalid V,
	check func(t *testing.T, impl T),
) MutatorAssertion[T, V] {
	return func(ctx MutatorContext[T, V]) {
		ctx.T.Run("rejects invalid", func(t *testing.T) {
			t.Parallel()
			impl := ctx.Factory()
			ctx.Call(t.Context(), impl, invalid)
			check(t, impl)
		})
	}
}

// AssertMutatorRejectInvalidWith calls the mutator on an invalid-factory
// impl with the sample value and asserts no panic. Mutators have no
// return position; the contract under "invalid impl" is the no-panic
// guarantee — observable rejection (the mutation didn't take effect)
// belongs to the cross-method invariant directives that pair the
// writer with a reader.
func AssertMutatorRejectInvalidWith[T, V any](
	invalidFactory func() T,
	sample V,
) MutatorAssertion[T, V] {
	return func(ctx MutatorContext[T, V]) {
		ctx.T.Run("rejects invalid", func(t *testing.T) {
			t.Parallel()
			impl := invalidFactory()
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("mutator on invalid impl must not panic, got: %v", r)
				}
			}()
			ctx.Call(t.Context(), impl, sample)
		})
	}
}

// AssertMutatorConcurrentSafe runs the mutator from N goroutines concurrently
// using the given sample value.
func AssertMutatorConcurrentSafe[T, V any](sample V, workers, iters int) MutatorAssertion[T, V] {
	return func(ctx MutatorContext[T, V]) {
		ctx.T.Run("concurrent safe", func(t *testing.T) {
			t.Parallel()
			impl := ctx.Factory()
			var wg sync.WaitGroup
			for range workers {
				wg.Go(func() {
					for range iters {
						ctx.Call(t.Context(), impl, sample)
					}
				})
			}
			wg.Wait()
		})
	}
}
