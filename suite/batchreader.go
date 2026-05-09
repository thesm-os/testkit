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

// BatchReaderContext provides a typed factory and call function to
// BatchReader-shape primitives. A BatchReader-shaped method has the
// signature `func(ctx?, ...K) ([]V, error)` — variadic-key fetch.
type BatchReaderContext[T any, K comparable, V any] struct {
	T *testing.T
	bindings.BatchReaderBindings[T, K, V]
}

// BatchReaderAssertion is a typed conformance primitive for
// BatchReader-shaped methods.
type BatchReaderAssertion[T any, K comparable, V any] func(BatchReaderContext[T, K, V])

// AssertBatchReaderReturnsAll calls the reader with the given keys and
// asserts the returned slice has the expected length and contents.
func AssertBatchReaderReturnsAll[T any, K comparable, V any](
	keys []K,
	want []V,
) BatchReaderAssertion[T, K, V] {
	return func(ctx BatchReaderContext[T, K, V]) {
		ctx.T.Run("returns all", func(t *testing.T) {
			t.Parallel()
			impl := ctx.Factory()
			got, err := ctx.Call(t.Context(), impl, keys)
			testkit.NoError(t, err, "batch reader must not error")
			testkit.Equal(t, len(got), len(want), "batch reader must return one value per key")
		})
	}
}

// AssertBatchReaderReturnsSentinel calls the reader with keys including
// an unknown one and asserts the configured sentinel is returned.
func AssertBatchReaderReturnsSentinel[T any, K comparable, V any](
	keys []K,
	sentinel error,
) BatchReaderAssertion[T, K, V] {
	return func(ctx BatchReaderContext[T, K, V]) {
		ctx.T.Run("returns sentinel", func(t *testing.T) {
			t.Parallel()
			impl := ctx.Factory()
			_, err := ctx.Call(t.Context(), impl, keys)
			testkit.ErrorIs(t, err, sentinel,
				"batch reader must return sentinel when batch contains unknown key")
		})
	}
}

// AssertBatchReaderConsistent calls the reader N times with the same keys
// and asserts the returned slices match in length across calls.
func AssertBatchReaderConsistent[T any, K comparable, V any](
	keys []K,
	n int,
) BatchReaderAssertion[T, K, V] {
	return func(ctx BatchReaderContext[T, K, V]) {
		ctx.T.Run("consistent reads", func(t *testing.T) {
			if n < 2 {
				t.Fatalf("AssertBatchReaderConsistent: n must be >= 2, got %d", n)
			}
			t.Parallel()
			impl := ctx.Factory()
			first, err := ctx.Call(t.Context(), impl, keys)
			testkit.NoError(t, err, "first read must not error")
			for i := 1; i < n; i++ {
				got, err := ctx.Call(t.Context(), impl, keys)
				testkit.NoError(t, err, "read must not error")
				testkit.Equal(t, len(got), len(first), "batch length must be consistent")
			}
		})
	}
}

// AssertBatchReaderRespectsContext invokes the reader with a cancelled
// context and asserts context.Canceled is returned.
func AssertBatchReaderRespectsContext[T any, K comparable, V any](
	keys []K,
) BatchReaderAssertion[T, K, V] {
	return func(ctx BatchReaderContext[T, K, V]) {
		ctx.T.Run("respects context", func(t *testing.T) {
			t.Parallel()
			impl := ctx.Factory()
			cctx, cancel := context.WithCancel(t.Context())
			cancel()
			_, err := ctx.Call(cctx, impl, keys)
			testkit.ErrorIs(t, err, context.Canceled,
				"batch reader must surface ctx.Canceled when called with a cancelled context")
		})
	}
}

// AssertBatchReaderConcurrentSafe runs the reader from N goroutines
// concurrently using the given keys.
func AssertBatchReaderConcurrentSafe[T any, K comparable, V any](
	keys []K,
	workers, iters int,
) BatchReaderAssertion[T, K, V] {
	return func(ctx BatchReaderContext[T, K, V]) {
		ctx.T.Run("concurrent safe", func(t *testing.T) {
			t.Parallel()
			impl := ctx.Factory()
			var wg sync.WaitGroup
			for range workers {
				wg.Go(func() {
					for range iters {
						_, _ = ctx.Call(t.Context(), impl, keys)
					}
				})
			}
			wg.Wait()
		})
	}
}

// AssertBatchReaderSmoke calls the reader once with the sample keys on
// a fresh impl. The subtest fails fast on panic, surfacing a broken
// Factory or a method that panics on bare invocation as one localized
// failure before any contract assertion runs.
func AssertBatchReaderSmoke[T any, K comparable, V any](sampleKeys []K) BatchReaderAssertion[T, K, V] {
	return func(ctx BatchReaderContext[T, K, V]) {
		ctx.T.Run("smoke", func(t *testing.T) {
			t.Parallel()
			impl := ctx.Factory()
			_, _ = ctx.Call(t.Context(), impl, sampleKeys)
		})
	}
}

// AssertBatchReaderBaseline runs the BatchReader-shape baseline: smoke,
// ReturnsAll(keys, vals), RespectsContext, Consistent over 3 calls, and
// ConcurrentSafe (4×10). Optional extras (e.g. ReturnsSentinel) run
// between consistency and concurrency.
func AssertBatchReaderBaseline[T any, K comparable, V any](
	sampleKeys []K,
	sampleVals []V,
	extra ...BatchReaderAssertion[T, K, V],
) BatchReaderAssertion[T, K, V] {
	return func(ctx BatchReaderContext[T, K, V]) {
		AssertBatchReaderSmoke[T, K, V](sampleKeys)(ctx)
		AssertBatchReaderReturnsAll[T, K, V](sampleKeys, sampleVals)(ctx)
		AssertBatchReaderRespectsContext[T, K, V](sampleKeys)(ctx)
		AssertBatchReaderConsistent[T, K, V](sampleKeys, 3)(ctx)
		for _, e := range extra {
			e(ctx)
		}
		AssertBatchReaderConcurrentSafe[T, K, V](sampleKeys, 4, 10)(ctx)
	}
}
