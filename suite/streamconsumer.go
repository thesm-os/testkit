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

// StreamConsumerContext provides a typed factory and call function to
// StreamConsumer-shape primitives. A StreamConsumer-shaped method has
// the signature `func(ctx, S) (V, error)` where S is interface-typed
// (e.g. io.Reader). Distinct from Reader: the input is a stream rather
// than a key.
type StreamConsumerContext[T, S, V any] struct {
	T *testing.T
	bindings.StreamConsumerBindings[T, S, V]
}

// StreamConsumerAssertion is a typed conformance primitive for
// StreamConsumer-shaped methods.
type StreamConsumerAssertion[T, S, V any] func(StreamConsumerContext[T, S, V])

// AssertStreamConsumerSucceeds calls the consumer with the given stream
// and asserts the returned value matches.
func AssertStreamConsumerSucceeds[T, S any, V comparable](
	stream S,
	want V,
) StreamConsumerAssertion[T, S, V] {
	return func(ctx StreamConsumerContext[T, S, V]) {
		ctx.T.Run("succeeds", func(t *testing.T) {
			t.Parallel()
			impl := ctx.Factory()
			got, err := ctx.Call(t.Context(), impl, stream)
			testkit.NoError(t, err, "stream consumer must not error on valid stream")
			testkit.Equal(t, got, want, "stream consumer must return expected value")
		})
	}
}

// AssertStreamConsumerRejectInvalid calls the consumer with an invalid
// stream and asserts the configured sentinel is returned.
func AssertStreamConsumerRejectInvalid[T, S, V any](
	invalid S,
	sentinel error,
) StreamConsumerAssertion[T, S, V] {
	return func(ctx StreamConsumerContext[T, S, V]) {
		ctx.T.Run("rejects invalid", func(t *testing.T) {
			t.Parallel()
			impl := ctx.Factory()
			_, err := ctx.Call(t.Context(), impl, invalid)
			testkit.ErrorIs(t, err, sentinel,
				"stream consumer must surface sentinel for invalid stream")
		})
	}
}

// AssertStreamConsumerConsistent calls the consumer N times with the
// same stream-factory output and asserts consistent return values.
func AssertStreamConsumerConsistent[T, S any, V comparable](
	streamFactory func() S,
	n int,
) StreamConsumerAssertion[T, S, V] {
	return func(ctx StreamConsumerContext[T, S, V]) {
		ctx.T.Run("consistent", func(t *testing.T) {
			if n < 2 {
				t.Fatalf("AssertStreamConsumerConsistent: n must be >= 2, got %d", n)
			}
			t.Parallel()
			impl := ctx.Factory()
			first, err := ctx.Call(t.Context(), impl, streamFactory())
			testkit.NoError(t, err, "first call must not error")
			for i := 1; i < n; i++ {
				got, err := ctx.Call(t.Context(), impl, streamFactory())
				testkit.NoError(t, err, "call must not error")
				testkit.Equal(t, got, first, "stream consumer must be consistent")
			}
		})
	}
}

// AssertStreamConsumerRespectsContext invokes the consumer with a
// cancelled context and asserts context.Canceled is returned.
func AssertStreamConsumerRespectsContext[T, S, V any](stream S) StreamConsumerAssertion[T, S, V] {
	return func(ctx StreamConsumerContext[T, S, V]) {
		ctx.T.Run("respects context", func(t *testing.T) {
			t.Parallel()
			impl := ctx.Factory()
			cctx, cancel := context.WithCancel(t.Context())
			cancel()
			_, err := ctx.Call(cctx, impl, stream)
			testkit.ErrorIs(t, err, context.Canceled,
				"stream consumer must surface ctx.Canceled when called with a cancelled context")
		})
	}
}

// AssertStreamConsumerConcurrentSafe runs the consumer from N goroutines
// concurrently.
func AssertStreamConsumerConcurrentSafe[T, S, V any](
	streamFactory func() S,
	workers, iters int,
) StreamConsumerAssertion[T, S, V] {
	return func(ctx StreamConsumerContext[T, S, V]) {
		ctx.T.Run("concurrent safe", func(t *testing.T) {
			t.Parallel()
			impl := ctx.Factory()
			var wg sync.WaitGroup
			for range workers {
				wg.Go(func() {
					for range iters {
						_, _ = ctx.Call(t.Context(), impl, streamFactory())
					}
				})
			}
			wg.Wait()
		})
	}
}
