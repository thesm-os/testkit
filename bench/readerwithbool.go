// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package bench

import (
	"testing"

	"go.thesmos.sh/testkit/bindings"
)

// ReaderWithBoolContext provides a typed factory and call function to
// ReaderWithBool-shape bench primitives.
type ReaderWithBoolContext[T any, K comparable, V any] struct {
	B *testing.B
	bindings.ReaderWithBoolBindings[T, K, V]
}

// ReaderWithBool is a typed bench primitive for ReaderWithBool-shaped methods.
type ReaderWithBool[T any, K comparable, V any] func(ReaderWithBoolContext[T, K, V])

// ReaderWithBoolHotPath measures read throughput against a known key.
func ReaderWithBoolHotPath[T any, K comparable, V any](key K) ReaderWithBool[T, K, V] {
	return func(ctx ReaderWithBoolContext[T, K, V]) {
		ctx.B.Run("hot-path", func(b *testing.B) {
			impl := ctx.Factory()
			bctx := b.Context()
			b.ResetTimer()
			b.ReportAllocs()
			for b.Loop() {
				_, _ = ctx.Call(bctx, impl, key)
			}
		})
	}
}

// ReaderWithBoolAllocsWithin measures allocations per read and fails
// the benchmark if allocs exceed maxAllocs.
func ReaderWithBoolAllocsWithin[T any, K comparable, V any](key K, maxAllocs int) ReaderWithBool[T, K, V] {
	return func(ctx ReaderWithBoolContext[T, K, V]) {
		ctx.B.Run("allocs-within", func(b *testing.B) {
			impl := ctx.Factory()
			bctx := b.Context()
			allocs := testing.AllocsPerRun(100, func() {
				_, _ = ctx.Call(bctx, impl, key)
			})
			if int(allocs) > maxAllocs {
				b.Fatalf("reader-with-bool allocs %d exceeds budget %d", int(allocs), maxAllocs)
			}
		})
	}
}
