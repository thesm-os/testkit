// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package testkit

import (
	"testing"
)

// BenchStreamContext provides a typed factory and call function to
// Stream-shape bench primitives.
type BenchStreamContext[T, V any] struct {
	B *testing.B
	StreamBindings[T, V]
}

// BenchStream is a typed bench primitive for StreamReader-shaped methods.
type BenchStream[T, V any] func(BenchStreamContext[T, V])

// BenchStreamHotPath measures the full iteration latency and allocation rate.
// Reports ns/op and allocs/op.
func BenchStreamHotPath[T, V any]() BenchStream[T, V] {
	return func(ctx BenchStreamContext[T, V]) {
		ctx.B.Run("iterate-all", func(b *testing.B) {
			impl := ctx.Factory()
			b.ResetTimer()
			b.ReportAllocs()
			for b.Loop() {
				for v, err := range ctx.Call(b.Context(), impl) {
					_ = v
					_ = err
				}
			}
		})
	}
}

// BenchStreamAllocsWithin measures allocations per full iteration and fails
// the benchmark if allocs exceed maxAllocs.
func BenchStreamAllocsWithin[T, V any](maxAllocs int) BenchStream[T, V] {
	return func(ctx BenchStreamContext[T, V]) {
		ctx.B.Run("allocs-within", func(b *testing.B) {
			impl := ctx.Factory()
			allocs := testing.AllocsPerRun(100, func() {
				for v, err := range ctx.Call(b.Context(), impl) {
					_ = v
					_ = err
				}
			})
			if int(allocs) > maxAllocs {
				b.Fatalf("stream allocs %d exceeds budget %d", int(allocs), maxAllocs)
			}
		})
	}
}
