// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package bench

import (
	"testing"

	"go.thesmos.sh/testkit/bindings"
)

// MutatorContext provides a typed factory and call function to
// Mutator-shape bench primitives.
type MutatorContext[T, V any] struct {
	B *testing.B
	bindings.MutatorBindings[T, V]
}

// Mutator is a typed bench primitive for Mutator-shaped methods.
type Mutator[T, V any] func(MutatorContext[T, V])

// MutatorHotPath measures mutator throughput with a sample value.
func MutatorHotPath[T, V any](sample V) Mutator[T, V] {
	return func(ctx MutatorContext[T, V]) {
		ctx.B.Run("hot-path", func(b *testing.B) {
			impl := ctx.Factory()
			bctx := b.Context()
			b.ResetTimer()
			b.ReportAllocs()
			for b.Loop() {
				ctx.Call(bctx, impl, sample)
			}
		})
	}
}

// MutatorAllocsWithin measures allocations per mutator call and fails
// the benchmark if allocs exceed maxAllocs.
func MutatorAllocsWithin[T, V any](sample V, maxAllocs int) Mutator[T, V] {
	return func(ctx MutatorContext[T, V]) {
		ctx.B.Run("allocs-within", func(b *testing.B) {
			impl := ctx.Factory()
			bctx := b.Context()
			allocs := testing.AllocsPerRun(100, func() {
				ctx.Call(bctx, impl, sample)
			})
			if int(allocs) > maxAllocs {
				b.Fatalf("mutator allocs %d exceeds budget %d", int(allocs), maxAllocs)
			}
		})
	}
}
