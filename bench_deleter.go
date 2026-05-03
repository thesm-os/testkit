// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package testkit

import (
	"fmt"
	"testing"
)

// BenchDeleterContext provides a typed factory and call function to
// Deleter-shape bench primitives.
type BenchDeleterContext[T any, K comparable] struct {
	B *testing.B
	DeleterBindings[T, K]
}

// BenchDeleter is a typed bench primitive for Deleter-shaped methods.
type BenchDeleter[T any, K comparable] func(BenchDeleterContext[T, K])

// BenchDeleterHotPath measures the single-goroutine delete latency and
// allocation rate for the given key. Reports ns/op and allocs/op.
func BenchDeleterHotPath[T any, K comparable](key K) BenchDeleter[T, K] {
	return func(ctx BenchDeleterContext[T, K]) {
		ctx.B.Run(fmt.Sprintf("hot-path/%v", key), func(b *testing.B) {
			impl := ctx.Factory()
			b.ResetTimer()
			b.ReportAllocs()
			for b.Loop() {
				_ = ctx.Call(b.Context(), impl, key)
			}
		})
	}
}

// BenchDeleterAllocsWithin measures allocations per delete and fails the
// benchmark if allocs exceed maxAllocs.
func BenchDeleterAllocsWithin[T any, K comparable](key K, maxAllocs int) BenchDeleter[T, K] {
	return func(ctx BenchDeleterContext[T, K]) {
		ctx.B.Run(fmt.Sprintf("allocs-within-%d/%v", maxAllocs, key), func(b *testing.B) {
			allocs := testing.AllocsPerRun(100, func() {
				impl := ctx.Factory()
				_ = ctx.Call(b.Context(), impl, key)
			})
			if int(allocs) > maxAllocs {
				b.Fatalf("deleter allocs %d exceeds budget %d", int(allocs), maxAllocs)
			}
		})
	}
}
