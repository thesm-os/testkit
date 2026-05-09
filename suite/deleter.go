// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package suite

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/bindings"
)

// DeleterContext provides a typed factory and call function to Deleter-shape
// primitives. A Deleter-shaped method has the signature func(ctx, K) error
// and requires the //testkit:deleter directive.
type DeleterContext[T any, K comparable] struct {
	T *testing.T
	bindings.DeleterBindings[T, K]
}

// DeleterAssertion is a typed conformance primitive for Deleter-shaped methods.
type DeleterAssertion[T any, K comparable] func(DeleterContext[T, K])

// AssertDeleteSucceeds deletes the given key and asserts no error.
// The consumer should pre-populate the key via the factory.
func AssertDeleteSucceeds[T any, K comparable](key K) DeleterAssertion[T, K] {
	return func(ctx DeleterContext[T, K]) {
		ctx.T.Run(fmt.Sprintf("delete succeeds %v", key), func(t *testing.T) {
			t.Parallel()
			impl := ctx.Factory()
			err := ctx.Call(t.Context(), impl, key)
			testkit.NoError(t, err, "delete must succeed for existing key")
		})
	}
}

// AssertDeleteIdempotent deletes the given key twice and asserts neither
// call returns an error (or both return the same error).
func AssertDeleteIdempotent[T any, K comparable](key K) DeleterAssertion[T, K] {
	return func(ctx DeleterContext[T, K]) {
		ctx.T.Run("delete idempotent", func(t *testing.T) {
			t.Parallel()
			impl := ctx.Factory()
			err1 := ctx.Call(t.Context(), impl, key)
			err2 := ctx.Call(t.Context(), impl, key)
			if err1 == nil && err2 == nil {
				return
			}
			if err1 != nil && err2 != nil && (errors.Is(err1, err2) || errors.Is(err2, err1)) {
				return
			}
			t.Errorf("delete must be idempotent: first=%v, second=%v", err1, err2)
		})
	}
}

// AssertDeleteReturnsNotFound deletes an unknown key and asserts the
// returned error matches one of the declared sentinels via
// [errors.Is]. Variadic so methods declaring multiple
// //testkit:errors entries pass the full set.
func AssertDeleteReturnsNotFound[T any, K comparable](
	unknown K,
	sentinels ...error,
) DeleterAssertion[T, K] {
	return func(ctx DeleterContext[T, K]) {
		ctx.T.Run(fmt.Sprintf("delete returns not-found for %v", unknown), func(t *testing.T) {
			t.Parallel()
			impl := ctx.Factory()
			err := ctx.Call(t.Context(), impl, unknown)
			assertSentinelMatch(t, err,
				"delete of unknown key must return sentinel", sentinels...)
		})
	}
}

// AssertDeleterRespectsContext invokes the deleter with an already-cancelled
// context and asserts the impl returns context.Canceled.
func AssertDeleterRespectsContext[T any, K comparable](key K) DeleterAssertion[T, K] {
	return func(ctx DeleterContext[T, K]) {
		ctx.T.Run("respects context", func(t *testing.T) {
			t.Parallel()
			impl := ctx.Factory()
			cctx, cancel := context.WithCancel(t.Context())
			cancel()
			err := ctx.Call(cctx, impl, key)
			testkit.ErrorIs(t, err, context.Canceled,
				"deleter must surface ctx.Canceled when called with a cancelled context")
		})
	}
}

// AssertDeleterConcurrentSafe runs the deleter from N goroutines concurrently
// using the given key.
func AssertDeleterConcurrentSafe[T any, K comparable](key K, workers, iters int) DeleterAssertion[T, K] {
	return func(ctx DeleterContext[T, K]) {
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

// AssertDeleterSmoke calls the deleter once with the sample key on a
// fresh impl. The subtest fails fast on panic, surfacing a broken
// Factory or a method that panics on bare invocation as one localized
// failure before any contract assertion runs.
func AssertDeleterSmoke[T any, K comparable](sampleKey K) DeleterAssertion[T, K] {
	return func(ctx DeleterContext[T, K]) {
		ctx.T.Run("smoke", func(t *testing.T) {
			t.Parallel()
			impl := ctx.Factory()
			_ = ctx.Call(t.Context(), impl, sampleKey)
		})
	}
}

// AssertDeleterBaseline runs the Deleter-shape baseline: smoke,
// DeleteSucceeds(key), RespectsContext, DeleteIdempotent, and
// ConcurrentSafe (4×10). Optional extras (e.g. DeleteReturnsNotFound)
// run between idempotency and concurrency.
func AssertDeleterBaseline[T any, K comparable](
	sampleKey K,
	extra ...DeleterAssertion[T, K],
) DeleterAssertion[T, K] {
	return func(ctx DeleterContext[T, K]) {
		AssertDeleterSmoke[T, K](sampleKey)(ctx)
		AssertDeleteSucceeds[T, K](sampleKey)(ctx)
		AssertDeleterRespectsContext[T, K](sampleKey)(ctx)
		AssertDeleteIdempotent[T, K](sampleKey)(ctx)
		for _, e := range extra {
			e(ctx)
		}
		AssertDeleterConcurrentSafe[T, K](sampleKey, 4, 10)(ctx)
	}
}
