// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package testkit

import (
	"testing"
)

// BenchAggregatorContext provides a typed factory and call function to
// Aggregator-shape bench primitives.
type BenchAggregatorContext[T, R any] struct {
	B *testing.B
	AggregatorBindings[T, R]
}

// BenchAggregator is a typed bench primitive for Aggregator-shaped methods.
type BenchAggregator[T, R any] func(BenchAggregatorContext[T, R])

// BenchAggregatorAllocsWithin measures allocations per aggregation and fails
// the benchmark if allocs exceed maxAllocs.
func BenchAggregatorAllocsWithin[T, R any](maxAllocs int) BenchAggregator[T, R] {
	return func(ctx BenchAggregatorContext[T, R]) {
		ctx.B.Run("allocs-within", func(b *testing.B) {
			impl := ctx.Factory()
			allocs := testing.AllocsPerRun(100, func() {
				_, _ = ctx.Call(b.Context(), impl)
			})
			if int(allocs) > maxAllocs {
				b.Fatalf("aggregator allocs %d exceeds budget %d", int(allocs), maxAllocs)
			}
		})
	}
}
