// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package bench

import (
	"testing"

	"go.thesmos.sh/testkit/bindings"
)

// PredicateContext provides a typed factory and call function to
// Predicate-shape bench primitives.
type PredicateContext[T any] struct {
	B *testing.B
	bindings.PredicateBindings[T]
}

// Predicate is a typed bench primitive for Predicate-shaped methods.
type Predicate[T any] func(PredicateContext[T])

// PredicateAllocsWithin measures allocations per predicate call and fails
// the benchmark if allocs exceed maxAllocs.
func PredicateAllocsWithin[T any](maxAllocs int) Predicate[T] {
	return func(ctx PredicateContext[T]) {
		ctx.B.Run("allocs-within", func(b *testing.B) {
			impl := ctx.Factory()
			allocs := testing.AllocsPerRun(100, func() {
				_ = ctx.Call(impl)
			})
			if int(allocs) > maxAllocs {
				b.Fatalf("predicate allocs %d exceeds budget %d", int(allocs), maxAllocs)
			}
		})
	}
}
