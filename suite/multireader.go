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

// MultiReaderContext provides a typed factory and call function to
// MultiReader-shape primitives. A MultiReader-shaped method has the
// signature `func(ctx?, K) (V1, V2, error)` — "get the entity +
// metadata" idioms.
type MultiReaderContext[T any, K comparable, V1, V2 any] struct {
	T *testing.T
	bindings.MultiReaderBindings[T, K, V1, V2]
}

// MultiReaderAssertion is a typed conformance primitive for
// MultiReader-shaped methods.
type MultiReaderAssertion[T any, K comparable, V1, V2 any] func(MultiReaderContext[T, K, V1, V2])

// AssertMultiReaderReturnsForKey calls the reader with a known key and
// asserts both return values match.
func AssertMultiReaderReturnsForKey[T any, K, V1, V2 comparable](
	key K,
	want1 V1,
	want2 V2,
) MultiReaderAssertion[T, K, V1, V2] {
	return func(ctx MultiReaderContext[T, K, V1, V2]) {
		ctx.T.Run("returns for key", func(t *testing.T) {
			t.Parallel()
			impl := ctx.Factory()
			got1, got2, err := ctx.Call(t.Context(), impl, key)
			testkit.NoError(t, err, "reader must not error for known key")
			testkit.Equal(t, got1, want1, "first return value must match")
			testkit.Equal(t, got2, want2, "second return value must match")
		})
	}
}

// AssertMultiReaderReturnsSentinel calls the reader with an unknown key
// and asserts the configured sentinel error is returned.
func AssertMultiReaderReturnsSentinel[T any, K comparable, V1, V2 any](
	unknown K,
	sentinel error,
) MultiReaderAssertion[T, K, V1, V2] {
	return func(ctx MultiReaderContext[T, K, V1, V2]) {
		ctx.T.Run("returns sentinel", func(t *testing.T) {
			t.Parallel()
			impl := ctx.Factory()
			_, _, err := ctx.Call(t.Context(), impl, unknown)
			testkit.ErrorIs(t, err, sentinel, "reader must return sentinel for unknown key")
		})
	}
}

// AssertMultiReaderConsistent calls the reader N times and asserts both
// return values match across calls.
func AssertMultiReaderConsistent[T any, K, V1, V2 comparable](
	key K,
	n int,
) MultiReaderAssertion[T, K, V1, V2] {
	return func(ctx MultiReaderContext[T, K, V1, V2]) {
		ctx.T.Run("consistent reads", func(t *testing.T) {
			if n < 2 {
				t.Fatalf("AssertMultiReaderConsistent: n must be >= 2, got %d", n)
			}
			t.Parallel()
			impl := ctx.Factory()
			first1, first2, err := ctx.Call(t.Context(), impl, key)
			testkit.NoError(t, err, "first read must not error")
			for i := 1; i < n; i++ {
				got1, got2, err := ctx.Call(t.Context(), impl, key)
				testkit.NoError(t, err, "read must not error")
				testkit.Equal(t, got1, first1, "first value must be consistent")
				testkit.Equal(t, got2, first2, "second value must be consistent")
			}
		})
	}
}

// AssertMultiReaderRespectsContext invokes the reader with a cancelled
// context and asserts context.Canceled is returned.
func AssertMultiReaderRespectsContext[T any, K comparable, V1, V2 any](
	key K,
) MultiReaderAssertion[T, K, V1, V2] {
	return func(ctx MultiReaderContext[T, K, V1, V2]) {
		ctx.T.Run("respects context", func(t *testing.T) {
			t.Parallel()
			impl := ctx.Factory()
			cctx, cancel := context.WithCancel(t.Context())
			cancel()
			_, _, err := ctx.Call(cctx, impl, key)
			testkit.ErrorIs(t, err, context.Canceled,
				"multi-reader must surface ctx.Canceled when called with a cancelled context")
		})
	}
}

// AssertMultiReaderConcurrentSafe runs the reader from N goroutines
// concurrently using the given key.
func AssertMultiReaderConcurrentSafe[T any, K comparable, V1, V2 any](
	key K,
	workers, iters int,
) MultiReaderAssertion[T, K, V1, V2] {
	return func(ctx MultiReaderContext[T, K, V1, V2]) {
		ctx.T.Run("concurrent safe", func(t *testing.T) {
			t.Parallel()
			impl := ctx.Factory()
			var wg sync.WaitGroup
			for range workers {
				wg.Go(func() {
					for range iters {
						_, _, _ = ctx.Call(t.Context(), impl, key)
					}
				})
			}
			wg.Wait()
		})
	}
}

// AssertMultiReaderSmoke calls the reader once with the sample key on
// a fresh impl. The subtest fails fast on panic, surfacing a broken
// Factory or a method that panics on bare invocation as one localized
// failure before any contract assertion runs.
func AssertMultiReaderSmoke[T any, K comparable, V1, V2 any](sampleKey K) MultiReaderAssertion[T, K, V1, V2] {
	return func(ctx MultiReaderContext[T, K, V1, V2]) {
		ctx.T.Run("smoke", func(t *testing.T) {
			t.Parallel()
			impl := ctx.Factory()
			_, _, _ = ctx.Call(t.Context(), impl, sampleKey)
		})
	}
}

// AssertMultiReaderBaseline runs the MultiReader-shape baseline: smoke,
// ReturnsForKey(key, v1, v2), RespectsContext, Consistent over 3 calls,
// and ConcurrentSafe (4×10). Optional extras (e.g. ReturnsSentinel) run
// between consistency and concurrency.
func AssertMultiReaderBaseline[T any, K, V1, V2 comparable](
	sampleKey K,
	sampleV1 V1,
	sampleV2 V2,
	extra ...MultiReaderAssertion[T, K, V1, V2],
) MultiReaderAssertion[T, K, V1, V2] {
	return func(ctx MultiReaderContext[T, K, V1, V2]) {
		AssertMultiReaderSmoke[T, K, V1, V2](sampleKey)(ctx)
		AssertMultiReaderReturnsForKey[T, K, V1, V2](sampleKey, sampleV1, sampleV2)(ctx)
		AssertMultiReaderRespectsContext[T, K, V1, V2](sampleKey)(ctx)
		AssertMultiReaderConsistent[T, K, V1, V2](sampleKey, 3)(ctx)
		for _, e := range extra {
			e(ctx)
		}
		AssertMultiReaderConcurrentSafe[T, K, V1, V2](sampleKey, 4, 10)(ctx)
	}
}
