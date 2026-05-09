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

// CompositeWriterContext provides a typed factory and call function to
// CompositeWriter-shape primitives. A CompositeWriter-shaped method has
// the signature `func(ctx?, K1, V) error` — namespaced stores, tagged
// caches, two-key indexes.
type CompositeWriterContext[T any, K1 comparable, V any] struct {
	T *testing.T
	bindings.CompositeWriterBindings[T, K1, V]
}

// CompositeWriterAssertion is a typed conformance primitive for
// CompositeWriter-shaped methods.
type CompositeWriterAssertion[T any, K1 comparable, V any] func(CompositeWriterContext[T, K1, V])

// AssertCompositeWriteSucceeds writes to the given (k1, value) pair and
// asserts no error.
func AssertCompositeWriteSucceeds[T any, K1 comparable, V any](
	k1 K1,
	sample V,
) CompositeWriterAssertion[T, K1, V] {
	return func(ctx CompositeWriterContext[T, K1, V]) {
		ctx.T.Run("write succeeds", func(t *testing.T) {
			t.Parallel()
			impl := ctx.Factory()
			err := ctx.Call(t.Context(), impl, k1, sample)
			testkit.NoError(t, err, "composite write must succeed")
		})
	}
}

// AssertCompositeWriteRejectInvalid writes an invalid value and asserts
// the configured sentinel is returned.
func AssertCompositeWriteRejectInvalid[T any, K1 comparable, V any](
	k1 K1,
	invalid V,
	sentinel error,
) CompositeWriterAssertion[T, K1, V] {
	return func(ctx CompositeWriterContext[T, K1, V]) {
		ctx.T.Run("rejects invalid", func(t *testing.T) {
			t.Parallel()
			impl := ctx.Factory()
			err := ctx.Call(t.Context(), impl, k1, invalid)
			testkit.ErrorIs(t, err, sentinel,
				"composite writer must surface sentinel for invalid value")
		})
	}
}

// AssertCompositeWriterIdempotent writes the same (k1, value) pair twice
// and asserts both calls succeed.
func AssertCompositeWriterIdempotent[T any, K1 comparable, V any](
	k1 K1,
	sample V,
) CompositeWriterAssertion[T, K1, V] {
	return func(ctx CompositeWriterContext[T, K1, V]) {
		ctx.T.Run("idempotent", func(t *testing.T) {
			t.Parallel()
			impl := ctx.Factory()
			err := ctx.Call(t.Context(), impl, k1, sample)
			testkit.NoError(t, err, "first write must succeed")
			err = ctx.Call(t.Context(), impl, k1, sample)
			testkit.NoError(t, err, "second write of same pair must succeed (idempotent)")
		})
	}
}

// AssertCompositeWriterRespectsContext invokes the writer with a
// cancelled context and asserts context.Canceled is returned.
func AssertCompositeWriterRespectsContext[T any, K1 comparable, V any](
	k1 K1,
	sample V,
) CompositeWriterAssertion[T, K1, V] {
	return func(ctx CompositeWriterContext[T, K1, V]) {
		ctx.T.Run("respects context", func(t *testing.T) {
			t.Parallel()
			impl := ctx.Factory()
			cctx, cancel := context.WithCancel(t.Context())
			cancel()
			err := ctx.Call(cctx, impl, k1, sample)
			testkit.ErrorIs(t, err, context.Canceled,
				"composite writer must surface ctx.Canceled when called with a cancelled context")
		})
	}
}

// AssertCompositeWriterConcurrentSafe runs the writer from N goroutines
// concurrently using the given pair.
func AssertCompositeWriterConcurrentSafe[T any, K1 comparable, V any](
	k1 K1,
	sample V,
	workers, iters int,
) CompositeWriterAssertion[T, K1, V] {
	return func(ctx CompositeWriterContext[T, K1, V]) {
		ctx.T.Run("concurrent safe", func(t *testing.T) {
			t.Parallel()
			impl := ctx.Factory()
			var wg sync.WaitGroup
			for range workers {
				wg.Go(func() {
					for range iters {
						_ = ctx.Call(t.Context(), impl, k1, sample)
					}
				})
			}
			wg.Wait()
		})
	}
}
