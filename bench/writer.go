// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package bench

import (
	"fmt"
	"testing"

	"go.thesmos.sh/testkit/bindings"
)

// WriterContext provides a typed factory and call function to
// Writer-shape bench primitives.
type WriterContext[T, V any] struct {
	B *testing.B
	bindings.WriterBindings[T, V]
}

// Writer is a typed bench primitive for Writer-shaped methods.
type Writer[T, V any] func(WriterContext[T, V])

// WriterHotPath measures the single-goroutine write latency and
// allocation rate for the given value. Reports ns/op and allocs/op.
func WriterHotPath[T, V any](sample V) Writer[T, V] {
	return func(ctx WriterContext[T, V]) {
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

// WriterAllocsWithin measures allocations per write and fails the
// benchmark if allocs exceed maxAllocs. Uses a fresh factory per
// AllocsPerRun invocation to avoid state accumulation from 101 writes.
func WriterAllocsWithin[T, V any](sample V, maxAllocs int) Writer[T, V] {
	return func(ctx WriterContext[T, V]) {
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
