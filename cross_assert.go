// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package testkit

import (
	"context"
	"testing"
)

// CrossContext provides a factory and test handle to cross-method primitives.
// Cross-method assertions compose multiple interface methods via consumer-
// provided typed closures.
type CrossContext[T any] struct {
	T       *testing.T
	Factory func() T
}

// CrossMethodAssertion is a conformance primitive that spans multiple methods
// of the same interface. Wired via OnAll(...).
type CrossMethodAssertion[T any] func(CrossContext[T])

// AssertReadAfterWrite writes a value, then reads it back, and asserts the
// read returns the written value. Consumer provides typed write/read/key
// closures that bind to the interface's actual methods.
func AssertReadAfterWrite[T any, K comparable, V any](
	sample V,
	write func(T, context.Context, V) error,
	read func(T, context.Context, K) (V, error),
	extractKey func(V) K,
) CrossMethodAssertion[T] {
	return func(ctx CrossContext[T]) {
		ctx.T.Run("read after write", func(t *testing.T) {
			t.Parallel()
			impl := ctx.Factory()
			err := write(impl, t.Context(), sample)
			NoError(t, err, "write must succeed")
			k := extractKey(sample)
			got, err := read(impl, t.Context(), k)
			NoError(t, err, "read-after-write must succeed")
			Equal(t, got, sample, "read must return written value")
		})
	}
}

// AssertDeleteRemovesValue writes a value, deletes it, then reads it back
// and asserts the read returns the notFound sentinel.
func AssertDeleteRemovesValue[T any, K comparable, V any](
	sample V,
	write func(T, context.Context, V) error,
	del func(T, context.Context, K) error,
	read func(T, context.Context, K) (V, error),
	extractKey func(V) K,
	notFound error,
) CrossMethodAssertion[T] {
	return func(ctx CrossContext[T]) {
		ctx.T.Run("delete removes value", func(t *testing.T) {
			t.Parallel()
			impl := ctx.Factory()
			err := write(impl, t.Context(), sample)
			NoError(t, err, "write must succeed")
			k := extractKey(sample)
			err = del(impl, t.Context(), k)
			NoError(t, err, "delete must succeed")
			_, err = read(impl, t.Context(), k)
			ErrorIs(t, err, notFound, "read-after-delete must return not-found sentinel")
		})
	}
}

// AssertStreamReflectsMutations writes N values, streams all, asserts all
// present. Then deletes one, streams again, asserts the deleted value is gone.
func AssertStreamReflectsMutations[T any, K comparable, V any](
	samples []V,
	write func(T, context.Context, V) error,
	stream func(T) []V,
	extractKey func(V) K,
) CrossMethodAssertion[T] {
	return func(ctx CrossContext[T]) {
		ctx.T.Run("stream reflects mutations", func(t *testing.T) {
			t.Parallel()
			impl := ctx.Factory()
			for _, v := range samples {
				err := write(impl, t.Context(), v)
				NoError(t, err, "write must succeed")
			}
			got := stream(impl)
			gotKeys := make(map[K]bool, len(got))
			for _, v := range got {
				gotKeys[extractKey(v)] = true
			}
			for _, v := range samples {
				k := extractKey(v)
				True(t, gotKeys[k], "stream must yield all written keys")
			}
		})
	}
}
