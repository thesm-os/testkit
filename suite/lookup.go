// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package suite

import (
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/bindings"
)

// LookupContext provides a typed factory and call function to
// Lookup-shape primitives. A Lookup-shaped method has the signature
// func(ctx, K) (R1, R2, bool) or func(K) (R1, R2, bool).
type LookupContext[T any, K comparable, V, R any] struct {
	T *testing.T
	bindings.LookupBindings[T, K, V, R]
}

// LookupAssertion is a typed conformance primitive for Lookup-shaped methods.
type LookupAssertion[T any, K comparable, V, R any] func(LookupContext[T, K, V, R])

// AssertLookupReturns calls the method with a known key and asserts
// ok is true and the first return value matches.
func AssertLookupReturns[T any, K comparable, V, R any](key K, want V) LookupAssertion[T, K, V, R] {
	return func(ctx LookupContext[T, K, V, R]) {
		ctx.T.Run("returns for key", func(t *testing.T) {
			t.Parallel()
			impl := ctx.Factory()
			got, _, ok := ctx.Call(t.Context(), impl, key)
			testkit.True(t, ok, "must return ok=true for known key")
			testkit.Equal(t, got, want, "must return expected value")
		})
	}
}

// AssertLookupMissing calls the method with an unknown key and asserts
// ok is false.
func AssertLookupMissing[T any, K comparable, V, R any](unknown K) LookupAssertion[T, K, V, R] {
	return func(ctx LookupContext[T, K, V, R]) {
		ctx.T.Run("missing key returns false", func(t *testing.T) {
			t.Parallel()
			impl := ctx.Factory()
			_, _, ok := ctx.Call(t.Context(), impl, unknown)
			testkit.False(t, ok, "must return ok=false for unknown key")
		})
	}
}
