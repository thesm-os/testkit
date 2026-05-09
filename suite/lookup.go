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

// LookupContext provides a typed factory and call function to
// Lookup-shape primitives. A Lookup-shaped method has the signature
// func(ctx, K) (R1, R2, bool) or func(K) (R1, R2, bool).
type LookupContext[T any, K comparable, V, R any] struct {
	T *testing.T
	bindings.LookupBindings[T, K, V, R]
}

// LookupAssertion is a typed conformance primitive for Lookup-shaped methods.
type LookupAssertion[T any, K comparable, V, R any] func(LookupContext[T, K, V, R])

// AssertLookupReturns calls the method with a known key and asserts
// ok is true and the first return value matches.
func AssertLookupReturns[T any, K comparable, V, R any](key K, want V) LookupAssertion[T, K, V, R] {
	return func(ctx LookupContext[T, K, V, R]) {
		ctx.T.Run("returns for key", func(t *testing.T) {
			t.Parallel()
			impl := ctx.Factory()
			got, _, ok := ctx.Call(t.Context(), impl, key)
			testkit.True(t, ok, "must return ok=true for known key")
			testkit.Equal(t, got, want, "must return expected value")
		})
	}
}

// AssertLookupMissing calls the method with an unknown key and asserts
// ok is false.
func AssertLookupMissing[T any, K comparable, V, R any](unknown K) LookupAssertion[T, K, V, R] {
	return func(ctx LookupContext[T, K, V, R]) {
		ctx.T.Run("missing key returns false", func(t *testing.T) {
			t.Parallel()
			impl := ctx.Factory()
			_, _, ok := ctx.Call(t.Context(), impl, unknown)
			testkit.False(t, ok, "must return ok=false for unknown key")
		})
	}
}

// AssertLookupConsistent calls the method N times with the same key and
// asserts all results agree.
func AssertLookupConsistent[T any, K comparable, V, R any](key K, n int) LookupAssertion[T, K, V, R] {
	return func(ctx LookupContext[T, K, V, R]) {
		ctx.T.Run("consistent reads", func(t *testing.T) {
			t.Parallel()
			impl := ctx.Factory()
			first1, first2, firstOK := ctx.Call(t.Context(), impl, key)
			for range n - 1 {
				got1, got2, ok := ctx.Call(t.Context(), impl, key)
				testkit.Equal(t, ok, firstOK, "ok must be consistent")
				if firstOK {
					testkit.Equal(t, got1, first1, "first value must be consistent")
					testkit.Equal(t, got2, first2, "second value must be consistent")
				}
			}
		})
	}
}

// AssertLookupRespectsContext invokes the method with a cancelled context
// and asserts (zero, zero, false). Lookup has no error position to surface
// ctx.Canceled, so the contract under cancellation is the miss outcome.
func AssertLookupRespectsContext[T any, K comparable, V, R any](key K) LookupAssertion[T, K, V, R] {
	return func(ctx LookupContext[T, K, V, R]) {
		ctx.T.Run("respects context", func(t *testing.T) {
			t.Parallel()
			impl := ctx.Factory()
			cctx, cancel := context.WithCancel(t.Context())
			cancel()
			_, _, ok := ctx.Call(cctx, impl, key)
			testkit.False(t, ok,
				"lookup must return ok=false when called with a cancelled context")
		})
	}
}

// AssertLookupConcurrentSafe runs the method from N goroutines concurrently
// using the given key.
func AssertLookupConcurrentSafe[T any, K comparable, V, R any](key K, workers, iters int) LookupAssertion[T, K, V, R] {
	return func(ctx LookupContext[T, K, V, R]) {
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

// AssertLookupSmoke calls the lookup once with the sample key on a
// fresh impl. The subtest fails fast on panic, surfacing a broken
// Factory or a method that panics on bare invocation as one localized
// failure before any contract assertion runs.
func AssertLookupSmoke[T any, K comparable, V, R any](sampleKey K) LookupAssertion[T, K, V, R] {
	return func(ctx LookupContext[T, K, V, R]) {
		ctx.T.Run("smoke", func(t *testing.T) {
			t.Parallel()
			impl := ctx.Factory()
			_, _, _ = ctx.Call(t.Context(), impl, sampleKey)
		})
	}
}

// AssertLookupBaseline runs the Lookup-shape baseline: smoke,
// Returns(key, want), RespectsContext, Consistent over 3 calls,
// Missing(zeroKey), and ConcurrentSafe (4×10). Optional extras run
// between the missing check and concurrency.
func AssertLookupBaseline[T any, K comparable, V, R any](
	sampleKey K,
	sampleVal V,
	zeroKey K,
	extra ...LookupAssertion[T, K, V, R],
) LookupAssertion[T, K, V, R] {
	return func(ctx LookupContext[T, K, V, R]) {
		AssertLookupSmoke[T, K, V, R](sampleKey)(ctx)
		AssertLookupReturns[T, K, V, R](sampleKey, sampleVal)(ctx)
		AssertLookupRespectsContext[T, K, V, R](sampleKey)(ctx)
		AssertLookupConsistent[T, K, V, R](sampleKey, 3)(ctx)
		AssertLookupMissing[T, K, V, R](zeroKey)(ctx)
		for _, e := range extra {
			e(ctx)
		}
		AssertLookupConcurrentSafe[T, K, V, R](sampleKey, 4, 10)(ctx)
	}
}
