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

// WriterContext provides typed domain inputs and a typed call function to
// Writer-shape primitives. Populated by generator-emitted options.
//
// A Writer-shaped method has the signature func(ctx, V) error
// or func(ctx, V) (R, error).
type WriterContext[T any, V any] struct {
	T *testing.T
	bindings.WriterBindings[T, V]
}

// WriterAssertion is a typed conformance primitive for Writer-shaped methods.
type WriterAssertion[T any, V any] func(WriterContext[T, V])

// AssertWriteSucceeds writes the given value and asserts no error.
func AssertWriteSucceeds[T, V any](sample V) WriterAssertion[T, V] {
	return func(ctx WriterContext[T, V]) {
		ctx.T.Run("write succeeds", func(t *testing.T) {
			t.Parallel()
			impl := ctx.Factory()
			err := ctx.Call(t.Context(), impl, sample)
			testkit.NoError(t, err, "write must succeed for sample value")
		})
	}
}

// AssertWriteIsObservable writes a value, then reads it back via a
// consumer-provided reader function, and asserts the value matches.
func AssertWriteIsObservable[T, V any, K comparable](
	sample V,
	extractKey func(V) K,
	reader func(context.Context, T, K) (V, error),
) WriterAssertion[T, V] {
	return func(ctx WriterContext[T, V]) {
		ctx.T.Run("write is observable", func(t *testing.T) {
			t.Parallel()
			impl := ctx.Factory()
			err := ctx.Call(t.Context(), impl, sample)
			testkit.NoError(t, err, "write must succeed")
			k := extractKey(sample)
			got, err := reader(t.Context(), impl, k)
			testkit.NoError(t, err, "read-back must succeed")
			testkit.Equal(t, got, sample, "read-back must return written value")
		})
	}
}

// AssertWriteRejectInvalid writes an invalid value and asserts error.
// If sentinel is non-nil, also asserts the error wraps it.
func AssertWriteRejectInvalid[T, V any](invalid V, sentinel error) WriterAssertion[T, V] {
	return func(ctx WriterContext[T, V]) {
		ctx.T.Run("write rejects invalid", func(t *testing.T) {
			t.Parallel()
			impl := ctx.Factory()
			err := ctx.Call(t.Context(), impl, invalid)
			testkit.Error(t, err, "write must reject invalid value")
			if sentinel != nil {
				testkit.ErrorIs(t, err, sentinel, "write must return expected sentinel")
			}
		})
	}
}

// AssertWriteOverwrite writes a value, then writes a second value with the
// same key, and asserts the second value is observable.
func AssertWriteOverwrite[T, V any, K comparable](
	first, second V,
	extractKey func(V) K,
	reader func(context.Context, T, K) (V, error),
) WriterAssertion[T, V] {
	return func(ctx WriterContext[T, V]) {
		ctx.T.Run("write overwrites", func(t *testing.T) {
			t.Parallel()
			impl := ctx.Factory()
			err := ctx.Call(t.Context(), impl, first)
			testkit.NoError(t, err, "first write must succeed")
			err = ctx.Call(t.Context(), impl, second)
			testkit.NoError(t, err, "second write must succeed")
			k := extractKey(second)
			got, err := reader(t.Context(), impl, k)
			testkit.NoError(t, err, "read-back must succeed")
			testkit.Equal(t, got, second, "read-back must return second (overwritten) value")
		})
	}
}

// AssertWriterRespectsContext invokes the writer with an already-cancelled
// context and asserts the impl returns context.Canceled.
func AssertWriterRespectsContext[T, V any](sample V) WriterAssertion[T, V] {
	return func(ctx WriterContext[T, V]) {
		ctx.T.Run("respects context", func(t *testing.T) {
			t.Parallel()
			impl := ctx.Factory()
			cctx, cancel := context.WithCancel(t.Context())
			cancel()
			err := ctx.Call(cctx, impl, sample)
			testkit.ErrorIs(t, err, context.Canceled,
				"writer must surface ctx.Canceled when called with a cancelled context")
		})
	}
}

// AssertWriterIdempotent writes the same value twice and asserts both calls
// succeed. The contract: writing the same value should be safe to repeat —
// transient retries, at-least-once delivery, redrive — without altering
// observable state beyond the first write's effect. Use with the Writer-
// shape Idempotent baseline; pair with [AssertWriteIsObservable] to assert
// the post-state matches across both writes.
func AssertWriterIdempotent[T, V any](sample V) WriterAssertion[T, V] {
	return func(ctx WriterContext[T, V]) {
		ctx.T.Run("idempotent", func(t *testing.T) {
			t.Parallel()
			impl := ctx.Factory()
			err := ctx.Call(t.Context(), impl, sample)
			testkit.NoError(t, err, "first write must succeed")
			err = ctx.Call(t.Context(), impl, sample)
			testkit.NoError(t, err, "second write of same value must succeed (idempotent)")
		})
	}
}

// AssertWriterConcurrentSafe runs the writer from N goroutines concurrently
// using the given sample value. The race detector finds data races when
// -race is enabled; panics propagate.
func AssertWriterConcurrentSafe[T, V any](
	sample V,
	workers, iters int,
) WriterAssertion[T, V] {
	return func(ctx WriterContext[T, V]) {
		ctx.T.Run("concurrent safe", func(t *testing.T) {
			t.Parallel()
			impl := ctx.Factory()
			var wg sync.WaitGroup
			for range workers {
				wg.Go(func() {
					for range iters {
						_ = ctx.Call(t.Context(), impl, sample)
					}
				})
			}
			wg.Wait()
		})
	}
}

// AssertWriterSmoke calls the writer once with the sample value on a
// fresh impl. The subtest fails fast on panic, surfacing a broken
// Factory or a method that panics on bare invocation as one localized
// failure before any contract assertion runs.
func AssertWriterSmoke[T, V any](sample V) WriterAssertion[T, V] {
	return func(ctx WriterContext[T, V]) {
		ctx.T.Run("smoke", func(t *testing.T) {
			t.Parallel()
			impl := ctx.Factory()
			_ = ctx.Call(t.Context(), impl, sample)
		})
	}
}

// AssertWriterBaseline runs the Writer-shape baseline: smoke,
// WriteSucceeds(sample), RespectsContext, Idempotent, and ConcurrentSafe
// (4×10). Optional extras (e.g. WriteRejectInvalid for methods that
// declare //testkit:errors with a nameable sentinel) run between
// idempotency and concurrency.
func AssertWriterBaseline[T, V any](
	sample V,
	extra ...WriterAssertion[T, V],
) WriterAssertion[T, V] {
	return func(ctx WriterContext[T, V]) {
		AssertWriterSmoke[T, V](sample)(ctx)
		AssertWriteSucceeds[T, V](sample)(ctx)
		AssertWriterRespectsContext[T, V](sample)(ctx)
		AssertWriterIdempotent[T, V](sample)(ctx)
		for _, e := range extra {
			e(ctx)
		}
		AssertWriterConcurrentSafe[T, V](sample, 4, 10)(ctx)
	}
}
