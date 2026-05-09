// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package suite

import (
	"sync"
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/bindings"
)

// ReaderNoErrorContext provides a typed factory and call function to
// ReaderNoError-shape primitives. A ReaderNoError-shaped method has the
// signature `func(ctx?, K) V` — infallible lookups against in-memory
// state (caches, gauges, stable mappings).
type ReaderNoErrorContext[T any, K comparable, V any] struct {
	T *testing.T
	bindings.ReaderNoErrorBindings[T, K, V]
}

// ReaderNoErrorAssertion is a typed conformance primitive for
// ReaderNoError-shaped methods.
type ReaderNoErrorAssertion[T any, K comparable, V any] func(ReaderNoErrorContext[T, K, V])

// AssertReaderNoErrorReturnsForKey calls the reader with a known key and
// asserts the value matches.
func AssertReaderNoErrorReturnsForKey[T any, K comparable, V any](
	key K,
	want V,
) ReaderNoErrorAssertion[T, K, V] {
	return func(ctx ReaderNoErrorContext[T, K, V]) {
		ctx.T.Run("returns for key", func(t *testing.T) {
			t.Parallel()
			impl := ctx.Factory()
			got := ctx.Call(t.Context(), impl, key)
			testkit.Equal(t, got, want, "reader must return expected value")
		})
	}
}

// AssertReaderNoErrorConsistent calls the reader N times with the same
// key and asserts all results are equal.
func AssertReaderNoErrorConsistent[T any, K, V comparable](
	key K,
	n int,
) ReaderNoErrorAssertion[T, K, V] {
	return func(ctx ReaderNoErrorContext[T, K, V]) {
		ctx.T.Run("consistent reads", func(t *testing.T) {
			if n < 2 {
				t.Fatalf("AssertReaderNoErrorConsistent: n must be >= 2, got %d", n)
			}
			t.Parallel()
			impl := ctx.Factory()
			first := ctx.Call(t.Context(), impl, key)
			for i := 1; i < n; i++ {
				got := ctx.Call(t.Context(), impl, key)
				testkit.Equal(t, got, first, "read must be consistent")
			}
		})
	}
}

// AssertReaderNoErrorZeroOnUnknown calls the reader with an unknown key
// and asserts the result equals the configured `zero` value (typically
// the type's zero, e.g. "" for string, 0 for int).
func AssertReaderNoErrorZeroOnUnknown[T any, K, V comparable](
	unknown K,
	zero V,
) ReaderNoErrorAssertion[T, K, V] {
	return func(ctx ReaderNoErrorContext[T, K, V]) {
		ctx.T.Run("zero on unknown key", func(t *testing.T) {
			t.Parallel()
			impl := ctx.Factory()
			got := ctx.Call(t.Context(), impl, unknown)
			testkit.Equal(t, got, zero, "reader must return zero on unknown key")
		})
	}
}

// AssertReaderNoErrorRespectsContext is the structural ctx-respect
// guarantee. ReaderNoError methods that take ctx surface a panic-free
// smoke under cancellation; methods without ctx have the guarantee
// signature-trivially. Either way, the call must not block.
func AssertReaderNoErrorRespectsContext[T any, K comparable, V any](
	key K,
) ReaderNoErrorAssertion[T, K, V] {
	return func(ctx ReaderNoErrorContext[T, K, V]) {
		ctx.T.Run("respects context", func(t *testing.T) {
			t.Parallel()
			impl := ctx.Factory()
			_ = ctx.Call(t.Context(), impl, key)
		})
	}
}

// AssertReaderNoErrorConcurrentSafe runs the reader from N goroutines
// concurrently using the given key.
func AssertReaderNoErrorConcurrentSafe[T any, K comparable, V any](
	key K,
	workers, iters int,
) ReaderNoErrorAssertion[T, K, V] {
	return func(ctx ReaderNoErrorContext[T, K, V]) {
		ctx.T.Run("concurrent safe", func(t *testing.T) {
			t.Parallel()
			impl := ctx.Factory()
			var wg sync.WaitGroup
			for range workers {
				wg.Go(func() {
					for range iters {
						_ = ctx.Call(t.Context(), impl, key)
					}
				})
			}
			wg.Wait()
		})
	}
}
