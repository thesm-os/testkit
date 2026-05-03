// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package bench

import (
	"testing"

	"go.thesmos.sh/testkit/bindings"
)

// AggregatorContext provides a typed factory and call function to
// Aggregator-shape bench primitives.
type AggregatorContext[T, R any] struct {
	B *testing.B
	bindings.AggregatorBindings[T, R]
}

// Aggregator is a typed bench primitive for Aggregator-shaped methods.
type Aggregator[T, R any] func(AggregatorContext[T, R])

// AggregatorAllocsWithin measures allocations per aggregation and fails
// the benchmark if allocs exceed maxAllocs.
func AggregatorAllocsWithin[T, R any](maxAllocs int) Aggregator[T, R] {
	return func(ctx AggregatorContext[T, R]) {
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
