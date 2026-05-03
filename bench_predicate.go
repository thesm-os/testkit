// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package testkit

import (
	"testing"
)

// BenchPredicateContext provides a typed factory and call function to
// Predicate-shape bench primitives.
type BenchPredicateContext[T any] struct {
	B *testing.B
	PredicateBindings[T]
}

// BenchPredicate is a typed bench primitive for Predicate-shaped methods.
type BenchPredicate[T any] func(BenchPredicateContext[T])

// BenchPredicateAllocsWithin measures allocations per predicate call and fails
// the benchmark if allocs exceed maxAllocs.
func BenchPredicateAllocsWithin[T any](maxAllocs int) BenchPredicate[T] {
	return func(ctx BenchPredicateContext[T]) {
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
