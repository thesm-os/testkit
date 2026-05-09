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

// PointerReaderContext provides a typed factory and call function to
// PointerReader-shape primitives. A PointerReader-shaped method has the
// signature `func(ctx?, K) *V` — the nil-on-miss idiom.
type PointerReaderContext[T any, K comparable, V any] struct {
	T *testing.T
	bindings.PointerReaderBindings[T, K, V]
}

// PointerReaderAssertion is a typed conformance primitive for
// PointerReader-shaped methods.
type PointerReaderAssertion[T any, K comparable, V any] func(PointerReaderContext[T, K, V])

// AssertPointerReaderReturnsForKey calls the reader with a known key and
// asserts the returned pointer is non-nil and points at the expected
// value. `want` is itself a pointer so the generator can pass through
// the natural sample literal (`&V{...}`) without dereferencing — the
// assertion compares dereferenced values internally.
func AssertPointerReaderReturnsForKey[T any, K, V comparable](
	key K,
	want *V,
) PointerReaderAssertion[T, K, V] {
	return func(ctx PointerReaderContext[T, K, V]) {
		ctx.T.Run("returns for key", func(t *testing.T) {
			t.Parallel()
			impl := ctx.Factory()
			got := ctx.Call(t.Context(), impl, key)
			testkit.True(t, got != nil, "reader must return non-nil pointer for known key")
			if got != nil && want != nil {
				testkit.Equal(t, *got, *want, "reader must return expected value")
			}
		})
	}
}

// AssertPointerReaderNilOnUnknown calls the reader with an unknown key
// and asserts the returned pointer is nil.
func AssertPointerReaderNilOnUnknown[T any, K comparable, V any](
	unknown K,
) PointerReaderAssertion[T, K, V] {
	return func(ctx PointerReaderContext[T, K, V]) {
		ctx.T.Run("nil on unknown key", func(t *testing.T) {
			t.Parallel()
			impl := ctx.Factory()
			got := ctx.Call(t.Context(), impl, unknown)
			testkit.True(t, got == nil, "reader must return nil for unknown key")
		})
	}
}

// AssertPointerReaderConsistent calls the reader N times and asserts the
// returned pointed-at values are equal across calls.
func AssertPointerReaderConsistent[T any, K, V comparable](
	key K,
	n int,
) PointerReaderAssertion[T, K, V] {
	return func(ctx PointerReaderContext[T, K, V]) {
		ctx.T.Run("consistent reads", func(t *testing.T) {
			if n < 2 {
				t.Fatalf("AssertPointerReaderConsistent: n must be >= 2, got %d", n)
			}
			t.Parallel()
			impl := ctx.Factory()
			first := ctx.Call(t.Context(), impl, key)
			testkit.True(t, first != nil, "first read must return non-nil")
			for i := 1; i < n; i++ {
				got := ctx.Call(t.Context(), impl, key)
				testkit.True(t, got != nil, "read must return non-nil")
				if first != nil && got != nil {
					testkit.Equal(t, *got, *first, "read must be consistent")
				}
			}
		})
	}
}

// AssertPointerReaderRespectsContext invokes the reader with a cancelled
// context and asserts the returned pointer is nil — the contract under
// cancellation is "miss-shaped".
func AssertPointerReaderRespectsContext[T any, K comparable, V any](
	key K,
) PointerReaderAssertion[T, K, V] {
	return func(ctx PointerReaderContext[T, K, V]) {
		ctx.T.Run("respects context", func(t *testing.T) {
			t.Parallel()
			impl := ctx.Factory()
			cctx, cancel := context.WithCancel(t.Context())
			cancel()
			got := ctx.Call(cctx, impl, key)
			testkit.True(t, got == nil,
				"pointer-reader must return nil when called with a cancelled context")
		})
	}
}

// AssertPointerReaderConcurrentSafe runs the reader from N goroutines
// concurrently using the given key.
func AssertPointerReaderConcurrentSafe[T any, K comparable, V any](
	key K,
	workers, iters int,
) PointerReaderAssertion[T, K, V] {
	return func(ctx PointerReaderContext[T, K, V]) {
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
