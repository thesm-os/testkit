// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package testkit

import (
	"context"
	"fmt"
	"sync"
	"testing"
)

// ReaderBindings holds the reusable shape wiring for a Reader-shaped method.
// Shared by suite (via ReaderContext) and future generators (bench, model).
type ReaderBindings[T any, K comparable, V any] struct {
	Factory func() T
	Call    func(context.Context, T, K) (V, error)
}

// ReaderContext provides a typed factory and call function to Reader-shape
// primitives. Populated by generator-emitted On<Method> dispatch.
//
// A Reader-shaped method has the signature func(ctx, K) (V, error).
type ReaderContext[T any, K comparable, V any] struct {
	T *testing.T
	ReaderBindings[T, K, V]
}

// ReaderAssertion is a typed conformance primitive for Reader-shaped methods.
type ReaderAssertion[T any, K comparable, V any] func(ReaderContext[T, K, V])

// AssertReturnsForKey calls the reader with the given key and asserts it
// returns the expected value.
func AssertReturnsForKey[T any, K comparable, V any](key K, want V) ReaderAssertion[T, K, V] {
	return func(ctx ReaderContext[T, K, V]) {
		ctx.T.Run(fmt.Sprintf("returns for key %v", key), func(t *testing.T) {
			t.Parallel()
			impl := ctx.Factory()
			got, err := ctx.Call(t.Context(), impl, key)
			NoError(t, err, "reader must not error for known key")
			Equal(t, got, want, "reader must return expected value")
		})
	}
}

// AssertReturnsSentinel calls the reader with the given unknown key and
// asserts it returns the given sentinel error.
func AssertReturnsSentinel[T any, K comparable, V any](unknown K, sentinel error) ReaderAssertion[T, K, V] {
	return func(ctx ReaderContext[T, K, V]) {
		ctx.T.Run(fmt.Sprintf("returns sentinel for %v", unknown), func(t *testing.T) {
			t.Parallel()
			impl := ctx.Factory()
			_, err := ctx.Call(t.Context(), impl, unknown)
			ErrorIs(t, err, sentinel, "reader must return sentinel for unknown key")
		})
	}
}

// AssertConsistentReads calls the reader N times with the given key and
// asserts all results are equal. N must be >= 2.
func AssertConsistentReads[T any, K comparable, V any](key K, n int) ReaderAssertion[T, K, V] {
	return func(ctx ReaderContext[T, K, V]) {
		ctx.T.Run("consistent reads", func(t *testing.T) {
			if n < 2 {
				t.Fatalf("AssertConsistentReads: n must be >= 2, got %d", n)
			}
			t.Parallel()
			impl := ctx.Factory()
			first, err := ctx.Call(t.Context(), impl, key)
			NoError(t, err, "first read must not error")
			for i := 1; i < n; i++ {
				got, err := ctx.Call(t.Context(), impl, key)
				NoError(t, err, "read must not error")
				Equal(t, got, first, "read must be consistent")
			}
		})
	}
}

// AssertReadsAreNonMutating calls observe before and after reading the given
// key, and asserts the observable state did not change.
func AssertReadsAreNonMutating[T any, K comparable, V any, S comparable](
	key K,
	observe func(context.Context, T) S,
) ReaderAssertion[T, K, V] {
	return func(ctx ReaderContext[T, K, V]) {
		ctx.T.Run("reads are non-mutating", func(t *testing.T) {
			t.Parallel()
			impl := ctx.Factory()
			before := observe(t.Context(), impl)
			_, _ = ctx.Call(t.Context(), impl, key)
			after := observe(t.Context(), impl)
			Equal(t, before, after, "read must not mutate observable state")
		})
	}
}

// AssertReaderConcurrentSafe runs the reader from N goroutines concurrently
// using the given key. Panics propagate (Go default); race detector finds
// data races when -race is enabled.
func AssertReaderConcurrentSafe[T any, K comparable, V any](
	key K,
	workers, iters int,
) ReaderAssertion[T, K, V] {
	return func(ctx ReaderContext[T, K, V]) {
		ctx.T.Run("concurrent safe", func(t *testing.T) {
			t.Parallel()
			impl := ctx.Factory()
			var wg sync.WaitGroup
			for range workers {
				wg.Go(func() {
					for range iters {
						_, _ = ctx.Call(t.Context(), impl, key)
					}
				})
			}
			wg.Wait()
		})
	}
}
