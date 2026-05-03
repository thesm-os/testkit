// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package testkit

import (
	"fmt"
	"testing"
)

// BenchWriterContext provides a typed factory and call function to
// Writer-shape bench primitives.
type BenchWriterContext[T, V any] struct {
	B *testing.B
	WriterBindings[T, V]
}

// BenchWriter is a typed bench primitive for Writer-shaped methods.
type BenchWriter[T, V any] func(BenchWriterContext[T, V])

// BenchWriterHotPath measures the single-goroutine write latency and
// allocation rate for the given value. Reports ns/op and allocs/op.
func BenchWriterHotPath[T, V any](sample V) BenchWriter[T, V] {
	return func(ctx BenchWriterContext[T, V]) {
		ctx.B.Run(fmt.Sprintf("hot-path/%v", sample), func(b *testing.B) {
			impl := ctx.Factory()
			b.ResetTimer()
			b.ReportAllocs()
			for b.Loop() {
				_ = ctx.Call(b.Context(), impl, sample)
			}
		})
	}
}

// BenchWriterAllocsWithin measures allocations per write and fails the
// benchmark if allocs exceed maxAllocs. Uses a fresh factory per
// AllocsPerRun invocation to avoid state accumulation from 101 writes.
func BenchWriterAllocsWithin[T, V any](sample V, maxAllocs int) BenchWriter[T, V] {
	return func(ctx BenchWriterContext[T, V]) {
		ctx.B.Run(fmt.Sprintf("allocs-within-%d/%v", maxAllocs, sample), func(b *testing.B) {
			allocs := testing.AllocsPerRun(100, func() {
				impl := ctx.Factory()
				_ = ctx.Call(b.Context(), impl, sample)
			})
			if int(allocs) > maxAllocs {
				b.Fatalf("writer allocs %d exceeds budget %d", int(allocs), maxAllocs)
			}
		})
	}
}
