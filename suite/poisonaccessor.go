// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package suite

import (
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/bindings"
)

// PoisonAccessorContext provides a typed factory and call function to
// PoisonAccessor-shape primitives. A PoisonAccessor-shaped method has
// the signature func() error (no ctx, no params).
type PoisonAccessorContext[T any] struct {
	T *testing.T
	bindings.PoisonAccessorBindings[T]
}

// PoisonAccessorAssertion is a typed conformance primitive for
// PoisonAccessor-shaped methods.
type PoisonAccessorAssertion[T any] func(PoisonAccessorContext[T])

// AssertPoisonAccessorNilOnFresh calls the accessor on a fresh impl
// and asserts it returns nil (no poison state).
func AssertPoisonAccessorNilOnFresh[T any]() PoisonAccessorAssertion[T] {
	return func(ctx PoisonAccessorContext[T]) {
		ctx.T.Run("nil on fresh", func(t *testing.T) {
			t.Parallel()
			impl := ctx.Factory()
			err := ctx.Call(impl)
			testkit.NoError(t, err, "fresh impl must not be poisoned")
		})
	}
}

// AssertPoisonAccessorConsistent calls the accessor twice and asserts
// both calls return the same result.
func AssertPoisonAccessorConsistent[T any]() PoisonAccessorAssertion[T] {
	return func(ctx PoisonAccessorContext[T]) {
		ctx.T.Run("consistent", func(t *testing.T) {
			t.Parallel()
			impl := ctx.Factory()
			err1 := ctx.Call(impl)
			err2 := ctx.Call(impl)
			testkit.Equal(t, err1 == nil, err2 == nil,
				"accessor must return consistent nil/non-nil")
		})
	}
}
