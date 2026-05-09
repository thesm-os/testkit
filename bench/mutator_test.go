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

func benchMutatorCtx(b *testing.B) bench.MutatorContext[*accumulator, int64] {
	b.Helper()
	return bench.MutatorContext[*accumulator, int64]{
		B: b,
		MutatorBindings: bindings.MutatorBindings[*accumulator, int64]{
			Factory: newAccumulator,
			Call: func(ctx context.Context, a *accumulator, v int64) {
				a.Add(ctx, v)
			},
		},
	}
}

func BenchmarkMutatorHotPath(b *testing.B) {
	ctx := benchMutatorCtx(b)
	bench.MutatorHotPath[*accumulator, int64](1)(ctx)
}

func BenchmarkMutatorAllocsWithin(b *testing.B) {
	ctx := benchMutatorCtx(b)
	bench.MutatorAllocsWithin[*accumulator, int64](1, 0)(ctx)
}

func BenchmarkMutatorLatencyWithin(b *testing.B) {
	ctx := benchMutatorCtx(b)
	bench.MutatorLatencyWithin[*accumulator, int64](1, 100*time.Millisecond)(ctx)
}

func BenchmarkMutatorConcurrentThroughput(b *testing.B) {
	ctx := benchMutatorCtx(b)
	bench.MutatorConcurrentThroughput[*accumulator, int64](1, 4)(ctx)
}
