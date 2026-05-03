// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package bench

import (
	"testing"

	"go.thesmos.sh/testkit/bindings"
)

// LifecycleContext provides a typed factory and call function to
// Lifecycle-shape bench primitives.
type LifecycleContext[T any] struct {
	B *testing.B
	bindings.LifecycleBindings[T]
}

// Lifecycle is a typed bench primitive for Lifecycle-shaped methods.
type Lifecycle[T any] func(LifecycleContext[T])

// LifecycleAllocsWithin measures allocations per lifecycle call and fails
// the benchmark if allocs exceed maxAllocs.
func LifecycleAllocsWithin[T any](maxAllocs int) Lifecycle[T] {
	return func(ctx LifecycleContext[T]) {
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
