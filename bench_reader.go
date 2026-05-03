// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package testkit

import (
	"fmt"
	"testing"
)

// BenchReaderContext provides a typed factory and call function to
// Reader-shape bench primitives. Populated by generator-emitted
// BenchOn<Method> dispatch.
type BenchReaderContext[T any, K comparable, V any] struct {
	B *testing.B
	ReaderBindings[T, K, V]
}

// BenchReader is a typed bench primitive for Reader-shaped methods.
// Primitives are either measurements (HotPath, ConcurrentThroughput)
// or gates (AllocsWithin).
type BenchReader[T any, K comparable, V any] func(BenchReaderContext[T, K, V])

// BenchReaderHotPath measures the single-goroutine read latency and
// allocation rate for the given key. Reports ns/op and allocs/op.
// Measurement only — does not fail the benchmark.
func BenchReaderHotPath[T any, K comparable, V any](key K) BenchReader[T, K, V] {
	return func(ctx BenchReaderContext[T, K, V]) {
		ctx.B.Run(fmt.Sprintf("hot-path/%v", key), func(b *testing.B) {
			impl := ctx.Factory()
			b.ResetTimer()
			b.ReportAllocs()
			for b.Loop() {
				_, _ = ctx.Call(b.Context(), impl, key)
			}
		})
	}
}

// BenchReaderAllocsWithin measures allocations per read and fails the
// benchmark if allocs exceed maxAllocs. Deterministic gate suitable for CI.
func BenchReaderAllocsWithin[T any, K comparable, V any](key K, maxAllocs int) BenchReader[T, K, V] {
	return func(ctx BenchReaderContext[T, K, V]) {
		ctx.B.Run(fmt.Sprintf("allocs-within-%d/%v", maxAllocs, key), func(b *testing.B) {
			impl := ctx.Factory()
			allocs := testing.AllocsPerRun(100, func() {
				_, _ = ctx.Call(b.Context(), impl, key)
			})
			if int(allocs) > maxAllocs {
				b.Fatalf("reader allocs %d exceeds budget %d", int(allocs), maxAllocs)
			}
		})
	}
}

// BenchReaderConcurrentThroughput measures read throughput under contention.
// Uses b.RunParallel for correct iteration scaling and goroutine lifecycle.
// Reports ns/op and allocs/op. Measurement only — does not fail the benchmark.
func BenchReaderConcurrentThroughput[T any, K comparable, V any](key K, parallelism int) BenchReader[T, K, V] {
	return func(ctx BenchReaderContext[T, K, V]) {
		ctx.B.Run(fmt.Sprintf("concurrent-%d/%v", parallelism, key), func(b *testing.B) {
			impl := ctx.Factory()
			b.SetParallelism(parallelism)
			b.ResetTimer()
			b.ReportAllocs()
			b.RunParallel(func(pb *testing.PB) {
				for pb.Next() {
					_, _ = ctx.Call(b.Context(), impl, key)
				}
			})
		})
	}
}
