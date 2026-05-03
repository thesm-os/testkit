// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package testkit

import (
	"context"
	"iter"
	"testing"
)

// StreamBindings holds the reusable shape wiring for a StreamReader-shaped method.
// Shared substrate for shape-driven generators.
type StreamBindings[T any, V any] struct {
	Factory func() T
	Call    func(context.Context, T) iter.Seq2[V, error]
}

// StreamContext provides typed domain inputs and a typed call function to
// Stream-shape primitives. Populated by generator-emitted options.
//
// A StreamReader-shaped method returns iter.Seq[V] or iter.Seq2[V, error].
type StreamContext[T any, V any] struct {
	T *testing.T
	StreamBindings[T, V]
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
				NoError(t, err, "stream must not yield error")
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
				NoError(t, err, "first yield must not error")
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
				NoError(t, err, "first iteration must not error")
			}
			for _, err := range ctx.Call(t.Context(), impl) {
				NoError(t, err, "second iteration must not error")
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
				NoError(t, err, "stream must not error")
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
				NoError(t, err, "stream must not error")
				k := extractKey(v)
				if seen[k] {
					t.Errorf("duplicate key in stream: %v", k)
				}
				seen[k] = true
			}
		})
	}
}
