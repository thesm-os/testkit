// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package bench

import (
	"testing"

	"go.thesmos.sh/testkit/bindings"
)

// LookupContext provides a typed factory and call function to
// Lookup-shape bench primitives.
type LookupContext[T any, K comparable, V, R any] struct {
	B *testing.B
	bindings.LookupBindings[T, K, V, R]
}

// Lookup is a typed bench primitive for Lookup-shaped methods.
type Lookup[T any, K comparable, V, R any] func(LookupContext[T, K, V, R])

// LookupHotPath measures lookup throughput against a known key.
func LookupHotPath[T any, K comparable, V, R any](key K) Lookup[T, K, V, R] {
	return func(ctx LookupContext[T, K, V, R]) {
		ctx.B.Run("hot-path", func(b *testing.B) {
			impl := ctx.Factory()
			bctx := b.Context()
			b.ResetTimer()
			b.ReportAllocs()
			for b.Loop() {
				_, _, _ = ctx.Call(bctx, impl, key)
			}
		})
	}
}

// LookupAllocsWithin measures allocations per lookup and fails
// the benchmark if allocs exceed maxAllocs.
func LookupAllocsWithin[T any, K comparable, V, R any](key K, maxAllocs int) Lookup[T, K, V, R] {
	return func(ctx LookupContext[T, K, V, R]) {
		ctx.B.Run("allocs-within", func(b *testing.B) {
			impl := ctx.Factory()
			bctx := b.Context()
			allocs := testing.AllocsPerRun(100, func() {
				_, _, _ = ctx.Call(bctx, impl, key)
			})
			if int(allocs) > maxAllocs {
				b.Fatalf("lookup allocs %d exceeds budget %d", int(allocs), maxAllocs)
			}
		})
	}
}
