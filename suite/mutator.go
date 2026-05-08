// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package suite

import (
	"testing"

	"go.thesmos.sh/testkit/bindings"
)

// MutatorContext provides a typed factory and call function to
// Mutator-shape primitives. A Mutator-shaped method has the
// signature func(ctx, V) with no return value.
type MutatorContext[T, V any] struct {
	T *testing.T
	bindings.MutatorBindings[T, V]
}

// MutatorAssertion is a typed conformance primitive for Mutator-shaped methods.
type MutatorAssertion[T, V any] func(MutatorContext[T, V])

// AssertMutatorSucceeds calls the mutator with a sample value and
// asserts it does not panic.
func AssertMutatorSucceeds[T, V any](sample V) MutatorAssertion[T, V] {
	return func(ctx MutatorContext[T, V]) {
		ctx.T.Run("mutator succeeds", func(t *testing.T) {
			t.Parallel()
			impl := ctx.Factory()
			ctx.Call(t.Context(), impl, sample)
		})
	}
}

// AssertMutatorIdempotent calls the mutator twice with the same value
// and asserts neither call panics.
func AssertMutatorIdempotent[T, V any](sample V) MutatorAssertion[T, V] {
	return func(ctx MutatorContext[T, V]) {
		ctx.T.Run("mutator idempotent", func(t *testing.T) {
			t.Parallel()
			impl := ctx.Factory()
			ctx.Call(t.Context(), impl, sample)
			ctx.Call(t.Context(), impl, sample)
		})
	}
}
