// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package suite

import (
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/bindings"
)

// ReaderWithBoolContext provides a typed factory and call function to
// ReaderWithBool-shape primitives. A ReaderWithBool-shaped method has
// the signature func(ctx, K) (V, bool) or func(K) (V, bool).
type ReaderWithBoolContext[T any, K comparable, V any] struct {
	T *testing.T
	bindings.ReaderWithBoolBindings[T, K, V]
}

// ReaderWithBoolAssertion is a typed conformance primitive for
// ReaderWithBool-shaped methods.
type ReaderWithBoolAssertion[T any, K comparable, V any] func(ReaderWithBoolContext[T, K, V])

// AssertReaderWithBoolReturns calls the method with a known key and
// asserts the value matches and ok is true.
func AssertReaderWithBoolReturns[T any, K comparable, V any](key K, want V) ReaderWithBoolAssertion[T, K, V] {
	return func(ctx ReaderWithBoolContext[T, K, V]) {
		ctx.T.Run("returns for key", func(t *testing.T) {
			t.Parallel()
			impl := ctx.Factory()
			got, ok := ctx.Call(t.Context(), impl, key)
			testkit.True(t, ok, "must return ok=true for known key")
			testkit.Equal(t, got, want, "must return expected value")
		})
	}
}

// AssertReaderWithBoolMissing calls the method with an unknown key and
// asserts ok is false.
func AssertReaderWithBoolMissing[T any, K comparable, V any](unknown K) ReaderWithBoolAssertion[T, K, V] {
	return func(ctx ReaderWithBoolContext[T, K, V]) {
		ctx.T.Run("missing key returns false", func(t *testing.T) {
			t.Parallel()
			impl := ctx.Factory()
			_, ok := ctx.Call(t.Context(), impl, unknown)
			testkit.False(t, ok, "must return ok=false for unknown key")
		})
	}
}

// AssertReaderWithBoolConsistent calls the method N times with the same
// key and asserts all results are equal.
func AssertReaderWithBoolConsistent[T any, K comparable, V any](key K, n int) ReaderWithBoolAssertion[T, K, V] {
	return func(ctx ReaderWithBoolContext[T, K, V]) {
		ctx.T.Run("consistent reads", func(t *testing.T) {
			t.Parallel()
			impl := ctx.Factory()
			first, firstOK := ctx.Call(t.Context(), impl, key)
			for range n - 1 {
				got, ok := ctx.Call(t.Context(), impl, key)
				testkit.Equal(t, ok, firstOK, "ok must be consistent")
				if firstOK {
					testkit.Equal(t, got, first, "value must be consistent")
				}
			}
		})
	}
}
