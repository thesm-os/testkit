// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package bench_test

import (
	"context"
	"testing"

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

func BenchmarkAggregatorAllocsWithin(b *testing.B) {
	ctx := benchAggregatorCtx(b, 42)
	bench.AggregatorAllocsWithin[*itemCounter, int](0)(ctx)
}
