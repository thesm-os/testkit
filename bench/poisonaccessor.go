// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package bench

import (
	"testing"

	"go.thesmos.sh/testkit/bindings"
)

// PoisonAccessorContext provides a typed factory and call function to
// PoisonAccessor-shape bench primitives.
type PoisonAccessorContext[T any] struct {
	B *testing.B
	bindings.PoisonAccessorBindings[T]
}

// PoisonAccessor is a typed bench primitive for PoisonAccessor-shaped methods.
type PoisonAccessor[T any] func(PoisonAccessorContext[T])

// PoisonAccessorAllocsWithin measures allocations per accessor call
// and fails the benchmark if allocs exceed maxAllocs.
func PoisonAccessorAllocsWithin[T any](maxAllocs int) PoisonAccessor[T] {
	return func(ctx PoisonAccessorContext[T]) {
		ctx.B.Run("allocs-within", func(b *testing.B) {
			impl := ctx.Factory()
			allocs := testing.AllocsPerRun(100, func() {
				_ = ctx.Call(impl)
			})
			if int(allocs) > maxAllocs {
				b.Fatalf("poison-accessor allocs %d exceeds budget %d", int(allocs), maxAllocs)
			}
		})
	}
}
