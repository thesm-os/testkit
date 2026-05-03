// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package testkit

import (
	"testing"
)

// BenchLifecycleContext provides a typed factory and call function to
// Lifecycle-shape bench primitives.
type BenchLifecycleContext[T any] struct {
	B *testing.B
	LifecycleBindings[T]
}

// BenchLifecycle is a typed bench primitive for Lifecycle-shaped methods.
type BenchLifecycle[T any] func(BenchLifecycleContext[T])

// BenchLifecycleAllocsWithin measures allocations per lifecycle call and fails
// the benchmark if allocs exceed maxAllocs.
func BenchLifecycleAllocsWithin[T any](maxAllocs int) BenchLifecycle[T] {
	return func(ctx BenchLifecycleContext[T]) {
		ctx.B.Run("allocs-within", func(b *testing.B) {
			impl := ctx.Factory()
			allocs := testing.AllocsPerRun(100, func() {
				_ = ctx.Call(b.Context(), impl)
			})
			if int(allocs) > maxAllocs {
				b.Fatalf("lifecycle allocs %d exceeds budget %d", int(allocs), maxAllocs)
			}
		})
	}
}
