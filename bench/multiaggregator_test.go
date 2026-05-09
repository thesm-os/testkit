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

func benchMultiAggregatorCtx(b *testing.B) bench.MultiAggregatorContext[*stats, int, int64] {
	b.Helper()
	return bench.MultiAggregatorContext[*stats, int, int64]{
		B: b,
		MultiAggregatorBindings: bindings.MultiAggregatorBindings[*stats, int, int64]{
			Factory: func() *stats { return newStats(3, 600) },
			Call: func(ctx context.Context, s *stats) (int, int64, error) {
				return s.Stats(ctx)
			},
		},
	}
}

func BenchmarkMultiAggregatorHotPath(b *testing.B) {
	ctx := benchMultiAggregatorCtx(b)
	bench.MultiAggregatorHotPath[*stats, int, int64]()(ctx)
}

func BenchmarkMultiAggregatorAllocsWithin(b *testing.B) {
	ctx := benchMultiAggregatorCtx(b)
	bench.MultiAggregatorAllocsWithin[*stats, int, int64](0)(ctx)
}

func BenchmarkMultiAggregatorLatencyWithin(b *testing.B) {
	ctx := benchMultiAggregatorCtx(b)
	bench.MultiAggregatorLatencyWithin[*stats, int, int64](100 * time.Millisecond)(ctx)
}

func BenchmarkMultiAggregatorConcurrentThroughput(b *testing.B) {
	ctx := benchMultiAggregatorCtx(b)
	bench.MultiAggregatorConcurrentThroughput[*stats, int, int64](4)(ctx)
}
