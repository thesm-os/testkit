// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package bench_test

import (
	"context"
	"testing"
	"time"

	"go.thesmos.sh/testkit/bench"
	"go.thesmos.sh/testkit/bindings"
)

func benchAggregatorCtx(b *testing.B, n int) bench.AggregatorContext[*itemCounter, int] {
	b.Helper()
	return bench.AggregatorContext[*itemCounter, int]{
		B: b,
		AggregatorBindings: bindings.AggregatorBindings[*itemCounter, int]{
			Factory: func() *itemCounter { return newItemCounter(n) },
			Call: func(ctx context.Context, c *itemCounter) (int, error) {
				return c.Count(ctx)
			},
		},
	}
}

func BenchmarkAggregatorHotPath(b *testing.B) {
	ctx := benchAggregatorCtx(b, 42)
	bench.AggregatorHotPath[*itemCounter, int]()(ctx)
}

func BenchmarkAggregatorAllocsWithin(b *testing.B) {
	ctx := benchAggregatorCtx(b, 42)
	bench.AggregatorAllocsWithin[*itemCounter, int](0)(ctx)
}

func BenchmarkAggregatorLatencyWithin(b *testing.B) {
	ctx := benchAggregatorCtx(b, 42)
	bench.AggregatorLatencyWithin[*itemCounter, int](100 * time.Millisecond)(ctx)
}

func BenchmarkAggregatorConcurrentThroughput(b *testing.B) {
	ctx := benchAggregatorCtx(b, 42)
	bench.AggregatorConcurrentThroughput[*itemCounter, int](4)(ctx)
}
