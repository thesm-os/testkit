// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package bench

import (
	"testing"

	"go.thesmos.sh/testkit/bindings"
)

// StreamContext provides a typed factory and call function to
// Stream-shape bench primitives.
type StreamContext[T, V any] struct {
	B *testing.B
	bindings.StreamBindings[T, V]
}

// Stream is a typed bench primitive for StreamReader-shaped methods.
type Stream[T, V any] func(StreamContext[T, V])

// StreamHotPath measures the full iteration latency and allocation rate.
// Reports ns/op and allocs/op.
func StreamHotPath[T, V any]() Stream[T, V] {
	return func(ctx StreamContext[T, V]) {
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

// StreamAllocsWithin measures allocations per full iteration and fails
// the benchmark if allocs exceed maxAllocs.
func StreamAllocsWithin[T, V any](maxAllocs int) Stream[T, V] {
	return func(ctx StreamContext[T, V]) {
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
