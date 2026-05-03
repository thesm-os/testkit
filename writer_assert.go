// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package testkit

import (
	"context"
	"testing"
)

// WriterBindings holds the reusable shape wiring for a Writer-shaped method.
// Shared substrate for shape-driven generators.
type WriterBindings[T any, V any] struct {
	Factory func() T
	Call    func(context.Context, T, V) error
}

// WriterContext provides typed domain inputs and a typed call function to
// Writer-shape primitives. Populated by generator-emitted options.
//
// A Writer-shaped method has the signature func(ctx, V) error
// or func(ctx, V) (R, error).
type WriterContext[T any, V any] struct {
	T *testing.T
	WriterBindings[T, V]
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
			NoError(t, err, "write must succeed for sample value")
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
			NoError(t, err, "write must succeed")
			k := extractKey(sample)
			got, err := reader(t.Context(), impl, k)
			NoError(t, err, "read-back must succeed")
			Equal(t, got, sample, "read-back must return written value")
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
			Error(t, err, "write must reject invalid value")
			if sentinel != nil {
				ErrorIs(t, err, sentinel, "write must return expected sentinel")
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
			NoError(t, err, "first write must succeed")
			err = ctx.Call(t.Context(), impl, second)
			NoError(t, err, "second write must succeed")
			k := extractKey(second)
			got, err := reader(t.Context(), impl, k)
			NoError(t, err, "read-back must succeed")
			Equal(t, got, second, "read-back must return second (overwritten) value")
		})
	}
}
