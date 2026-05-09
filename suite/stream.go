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

// StreamContext provides typed domain inputs and a typed call function to
// Stream-shape primitives. Populated by generator-emitted options.
//
// A StreamReader-shaped method returns iter.Seq[V] or iter.Seq2[V, error].
type StreamContext[T any, V any] struct {
	T *testing.T
	bindings.StreamBindings[T, V]
}

// StreamAssertion is a typed conformance primitive for StreamReader-shaped methods.
type StreamAssertion[T any, V any] func(StreamContext[T, V])

// AssertStreamCompletes iterates the full stream and asserts no error is yielded.
func AssertStreamCompletes[T, V any]() StreamAssertion[T, V] {
	return func(ctx StreamContext[T, V]) {
		ctx.T.Run("stream completes", func(t *testing.T) {
			t.Parallel()
			impl := ctx.Factory()
			for _, err := range ctx.Call(t.Context(), impl) {
				testkit.NoError(t, err, "stream must not yield error")
			}
		})
	}
}

// AssertStreamRespectsBreak breaks the iteration after the first yield and
// verifies the break is legal (no error reported on the broken yield).
// Panics inside range propagate to the test (Go default).
func AssertStreamRespectsBreak[T, V any]() StreamAssertion[T, V] {
	return func(ctx StreamContext[T, V]) {
		ctx.T.Run("stream respects break", func(t *testing.T) {
			t.Parallel()
			impl := ctx.Factory()
			for _, err := range ctx.Call(t.Context(), impl) {
				testkit.NoError(t, err, "first yield must not error")
				break
			}
		})
	}
}

// AssertStreamReentrant iterates the stream twice and asserts both
// iterations complete without error.
func AssertStreamReentrant[T, V any]() StreamAssertion[T, V] {
	return func(ctx StreamContext[T, V]) {
		ctx.T.Run("stream reentrant", func(t *testing.T) {
			t.Parallel()
			impl := ctx.Factory()
			for _, err := range ctx.Call(t.Context(), impl) {
				testkit.NoError(t, err, "first iteration must not error")
			}
			for _, err := range ctx.Call(t.Context(), impl) {
				testkit.NoError(t, err, "second iteration must not error")
			}
		})
	}
}

// AssertStreamYieldsInOrder asserts that yielded values satisfy the given
// ordering predicate. Requires at least 2 items.
func AssertStreamYieldsInOrder[T, V any](
	less func(a, b V) bool,
) StreamAssertion[T, V] {
	return func(ctx StreamContext[T, V]) {
		ctx.T.Run("stream yields in order", func(t *testing.T) {
			t.Parallel()
			impl := ctx.Factory()
			var prev V
			i := 0
			for v, err := range ctx.Call(t.Context(), impl) {
				testkit.NoError(t, err, "stream must not error")
				if i > 0 && !less(prev, v) {
					t.Errorf("stream order violated at position %d", i)
				}
				prev = v
				i++
			}
			if i < 2 {
				t.Fatalf("AssertStreamYieldsInOrder: stream yielded %d items, requires >= 2", i)
			}
		})
	}
}

// AssertStreamHasNoDuplicates asserts that yielded values have distinct keys.
func AssertStreamHasNoDuplicates[T, V any, K comparable](
	extractKey func(V) K,
) StreamAssertion[T, V] {
	return func(ctx StreamContext[T, V]) {
		ctx.T.Run("stream has no duplicates", func(t *testing.T) {
			t.Parallel()
			impl := ctx.Factory()
			seen := make(map[K]bool)
			for v, err := range ctx.Call(t.Context(), impl) {
				testkit.NoError(t, err, "stream must not error")
				k := extractKey(v)
				if seen[k] {
					t.Errorf("duplicate key in stream: %v", k)
				}
				seen[k] = true
			}
		})
	}
}

// AssertStreamRespectsContext starts iteration, cancels the context after
// the first yield, and asserts subsequent yields surface context.Canceled
// (or terminate cleanly). A stream impl that ignores ctx mid-iteration is
// a real bug class — the impl yields stale data after the caller has
// signalled cancellation.
func AssertStreamRespectsContext[T, V any]() StreamAssertion[T, V] {
	return func(ctx StreamContext[T, V]) {
		ctx.T.Run("respects context (mid-stream)", func(t *testing.T) {
			t.Parallel()
			impl := ctx.Factory()
			cctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			seen := 0
			for _, err := range ctx.Call(cctx, impl) {
				if seen == 0 {
					cancel()
					seen++
					continue
				}
				if err != nil {
					testkit.ErrorIs(t, err, context.Canceled,
						"stream must surface ctx.Canceled after mid-stream cancel")
					return
				}
				seen++
				if seen > 100 {
					t.Fatalf("stream did not honor ctx-cancel within 100 yields")
				}
			}
		})
	}
}

// AssertStreamConcurrentSafe iterates the stream from N goroutines
// concurrently. Stream impls that share state across iterators trip the
// race detector under -race.
func AssertStreamConcurrentSafe[T, V any](workers int) StreamAssertion[T, V] {
	return func(ctx StreamContext[T, V]) {
		ctx.T.Run("concurrent safe", func(t *testing.T) {
			t.Parallel()
			impl := ctx.Factory()
			var wg sync.WaitGroup
			for range workers {
				wg.Go(func() {
					for _, err := range ctx.Call(t.Context(), impl) {
						_ = err
					}
				})
			}
			wg.Wait()
		})
	}
}
