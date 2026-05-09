// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package suite

import (
	"sync"
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/bindings"
)

// PoisonAccessorContext provides a typed factory and call function to
// PoisonAccessor-shape primitives. A PoisonAccessor-shaped method has
// the signature func() error (no ctx, no params).
type PoisonAccessorContext[T any] struct {
	T *testing.T
	bindings.PoisonAccessorBindings[T]
}

// PoisonAccessorAssertion is a typed conformance primitive for
// PoisonAccessor-shaped methods.
type PoisonAccessorAssertion[T any] func(PoisonAccessorContext[T])

// AssertPoisonAccessorNilOnFresh calls the accessor on a fresh impl
// and asserts it returns nil (no poison state).
func AssertPoisonAccessorNilOnFresh[T any]() PoisonAccessorAssertion[T] {
	return func(ctx PoisonAccessorContext[T]) {
		ctx.T.Run("nil on fresh", func(t *testing.T) {
			t.Parallel()
			impl := ctx.Factory()
			err := ctx.Call(impl)
			testkit.NoError(t, err, "fresh impl must not be poisoned")
		})
	}
}

// AssertPoisonAccessorConsistent calls the accessor twice and asserts
// both calls return the same result.
func AssertPoisonAccessorConsistent[T any]() PoisonAccessorAssertion[T] {
	return func(ctx PoisonAccessorContext[T]) {
		ctx.T.Run("consistent", func(t *testing.T) {
			t.Parallel()
			impl := ctx.Factory()
			err1 := ctx.Call(impl)
			err2 := ctx.Call(impl)
			testkit.Equal(t, err1 == nil, err2 == nil,
				"accessor must return consistent nil/non-nil")
		})
	}
}

// AssertPoisonAccessorRespectsContext is the structural ctx-respect
// guarantee for PoisonAccessor-shape methods. PoisonAccessor takes no
// context — cancellation is satisfied by signature.
func AssertPoisonAccessorRespectsContext[T any]() PoisonAccessorAssertion[T] {
	return func(ctx PoisonAccessorContext[T]) {
		ctx.T.Run("respects context (structural)", func(t *testing.T) {
			t.Parallel()
			impl := ctx.Factory()
			_ = ctx.Call(impl)
		})
	}
}

// AssertPoisonAccessorRejectInvalid asserts that a poisoned impl returns
// the configured sentinel. The consumer supplies a poisonedFactory that
// produces an impl in a known-bad state; the contract is that the
// accessor surfaces the sentinel rather than masking the poison.
func AssertPoisonAccessorRejectInvalid[T any](poisonedFactory func() T, sentinel error) PoisonAccessorAssertion[T] {
	return func(ctx PoisonAccessorContext[T]) {
		ctx.T.Run("rejects invalid (poisoned)", func(t *testing.T) {
			t.Parallel()
			impl := poisonedFactory()
			err := ctx.Call(impl)
			testkit.ErrorIs(t, err, sentinel,
				"poisoned impl must surface the configured sentinel")
		})
	}
}

// AssertPoisonAccessorConcurrentSafe runs the accessor from N goroutines
// concurrently.
func AssertPoisonAccessorConcurrentSafe[T any](workers, iters int) PoisonAccessorAssertion[T] {
	return func(ctx PoisonAccessorContext[T]) {
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
