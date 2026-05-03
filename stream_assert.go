// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package testkit

import (
	"iter"
	"testing"
)

// StreamContext provides typed domain inputs and a typed call function to
// Stream-shape primitives. Populated by generator-emitted options.
//
// A StreamReader-shaped method returns iter.Seq[V] or iter.Seq2[V, error].
type StreamContext[T any, V any] struct {
	T       *testing.T
	Factory func() T
	// Call returns the iterator. For iter.Seq2[V, error], the consumer
	// wraps the call to return iter.Seq2[V, error] directly.
	Call func(T) iter.Seq2[V, error]
}

// StreamAssertion is a typed conformance primitive for StreamReader-shaped methods.
type StreamAssertion[T any, V any] func(StreamContext[T, V])

// AssertStreamCompletes iterates the full stream and asserts no error is yielded.
func AssertStreamCompletes[T, V any]() StreamAssertion[T, V] {
	return func(ctx StreamContext[T, V]) {
		ctx.T.Run("stream completes", func(t *testing.T) {
			t.Parallel()
			impl := ctx.Factory()
			for _, err := range ctx.Call(impl) {
				NoError(t, err, "stream must not yield error")
			}
		})
	}
}

// AssertStreamRespectsBreak breaks the iterator after the first item and
// asserts no panic or resource leak.
func AssertStreamRespectsBreak[T, V any]() StreamAssertion[T, V] {
	return func(ctx StreamContext[T, V]) {
		ctx.T.Run("stream respects break", func(t *testing.T) {
			t.Parallel()
			impl := ctx.Factory()
			for _, err := range ctx.Call(impl) {
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
			for _, err := range ctx.Call(impl) {
				NoError(t, err, "first iteration must not error")
			}
			for _, err := range ctx.Call(impl) {
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
			for v, err := range ctx.Call(impl) {
				NoError(t, err, "stream must not error")
				if i > 0 && !less(prev, v) {
					t.Errorf("stream order violated at position %d", i)
				}
				prev = v
				i++
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
			for v, err := range ctx.Call(impl) {
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
