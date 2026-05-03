// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package testkit

import (
	"context"
	"testing"
)

// CrossBindings holds the reusable shape wiring for cross-method assertions.
// Shared by suite (via CrossContext) and future generators (bench, model).
type CrossBindings[T any] struct {
	Factory func() T
}

// CrossContext provides a factory and test handle to cross-method primitives.
// Cross-method assertions compose multiple interface methods via consumer-
// provided typed closures. PrePopulate is not applied — cross-method
// primitives are self-sufficient and manage their own state.
type CrossContext[T any] struct {
	T *testing.T
	CrossBindings[T]
}

// CrossMethodAssertion is a conformance primitive that spans multiple methods
// of the same interface. Wired via OnAll(...).
type CrossMethodAssertion[T any] func(CrossContext[T])

// AssertReadAfterWrite writes a value, then reads it back, and asserts the
// read returns the written value.
func AssertReadAfterWrite[T any, K comparable, V any](
	sample V,
	write func(context.Context, T, V) error,
	read func(context.Context, T, K) (V, error),
	extractKey func(V) K,
) CrossMethodAssertion[T] {
	return func(ctx CrossContext[T]) {
		ctx.T.Run("read after write", func(t *testing.T) {
			t.Parallel()
			impl := ctx.Factory()
			err := write(t.Context(), impl, sample)
			NoError(t, err, "write must succeed")
			k := extractKey(sample)
			got, err := read(t.Context(), impl, k)
			NoError(t, err, "read-after-write must succeed")
			Equal(t, got, sample, "read must return written value")
		})
	}
}

// AssertDeleteRemovesValue writes a value, deletes it, then reads it back
// and asserts the read returns the notFound sentinel.
func AssertDeleteRemovesValue[T any, K comparable, V any](
	sample V,
	write func(context.Context, T, V) error,
	del func(context.Context, T, K) error,
	read func(context.Context, T, K) (V, error),
	extractKey func(V) K,
	notFound error,
) CrossMethodAssertion[T] {
	return func(ctx CrossContext[T]) {
		ctx.T.Run("delete removes value", func(t *testing.T) {
			t.Parallel()
			impl := ctx.Factory()
			err := write(t.Context(), impl, sample)
			NoError(t, err, "write must succeed")
			k := extractKey(sample)
			err = del(t.Context(), impl, k)
			NoError(t, err, "delete must succeed")
			_, err = read(t.Context(), impl, k)
			ErrorIs(t, err, notFound, "read-after-delete must return not-found sentinel")
		})
	}
}

// AssertStreamReflectsMutations writes N values, streams all and asserts
// every written key is yielded. Then deletes one, streams again, and asserts
// the deleted key is absent.
func AssertStreamReflectsMutations[T any, K comparable, V any](
	samples []V,
	write func(context.Context, T, V) error,
	del func(context.Context, T, K) error,
	stream func(context.Context, T) []V,
	extractKey func(V) K,
) CrossMethodAssertion[T] {
	return func(ctx CrossContext[T]) {
		ctx.T.Run("stream reflects mutations", func(t *testing.T) {
			t.Parallel()
			impl := ctx.Factory()

			for _, v := range samples {
				err := write(t.Context(), impl, v)
				NoError(t, err, "write must succeed")
			}

			got := stream(t.Context(), impl)
			gotKeys := make(map[K]bool, len(got))
			for _, v := range got {
				gotKeys[extractKey(v)] = true
			}
			for _, v := range samples {
				k := extractKey(v)
				True(t, gotKeys[k], "stream must yield all written keys")
			}

			if len(samples) > 0 {
				delKey := extractKey(samples[0])
				err := del(t.Context(), impl, delKey)
				NoError(t, err, "delete must succeed")

				got2 := stream(t.Context(), impl)
				got2Keys := make(map[K]bool, len(got2))
				for _, v := range got2 {
					got2Keys[extractKey(v)] = true
				}
				False(t, got2Keys[delKey], "stream must not yield deleted key")
			}
		})
	}
}
