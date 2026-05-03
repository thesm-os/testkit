// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package testkit_test

import (
	"context"
	"testing"

	"go.thesmos.sh/testkit"
)

func benchAggregatorCtx(b *testing.B, n int) testkit.BenchAggregatorContext[*itemCounter, int] {
	b.Helper()
	return testkit.BenchAggregatorContext[*itemCounter, int]{
		B: b,
		AggregatorBindings: testkit.AggregatorBindings[*itemCounter, int]{
			Factory: func() *itemCounter { return newItemCounter(n) },
			Call: func(ctx context.Context, c *itemCounter) (int, error) {
				return c.Count(ctx)
			},
		},
	}
}

func BenchmarkAggregatorAllocsWithin(b *testing.B) {
	ctx := benchAggregatorCtx(b, 42)
	testkit.BenchAggregatorAllocsWithin[*itemCounter, int](0)(ctx)
}
