// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package testkit

import (
	"testing"
)

// BenchPureContext provides a typed factory and call function to
// Pure-shape bench primitives.
type BenchPureContext[T, R any] struct {
	B *testing.B
	PureBindings[T, R]
}

// BenchPure is a typed bench primitive for Pure-shaped methods.
type BenchPure[T, R any] func(BenchPureContext[T, R])

// BenchPureAllocsWithin measures allocations per pure call and fails
// the benchmark if allocs exceed maxAllocs.
func BenchPureAllocsWithin[T, R any](maxAllocs int) BenchPure[T, R] {
	return func(ctx BenchPureContext[T, R]) {
		ctx.B.Run("allocs-within", func(b *testing.B) {
			impl := ctx.Factory()
			allocs := testing.AllocsPerRun(100, func() {
				_ = ctx.Call(impl)
			})
			if int(allocs) > maxAllocs {
				b.Fatalf("pure allocs %d exceeds budget %d", int(allocs), maxAllocs)
			}
		})
	}
}
