// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package bench_test

import (
	"testing"
	"time"

	"go.thesmos.sh/testkit/bench"
	"go.thesmos.sh/testkit/bindings"
)

func benchPoisonAccessorCtx(b *testing.B) bench.PoisonAccessorContext[*healthChecker] {
	b.Helper()
	return bench.PoisonAccessorContext[*healthChecker]{
		B: b,
		PoisonAccessorBindings: bindings.PoisonAccessorBindings[*healthChecker]{
			Factory: newHealthChecker,
			Call:    func(h *healthChecker) error { return h.Err() },
		},
	}
}

func BenchmarkPoisonAccessorHotPath(b *testing.B) {
	ctx := benchPoisonAccessorCtx(b)
	bench.PoisonAccessorHotPath[*healthChecker]()(ctx)
}

func BenchmarkPoisonAccessorAllocsWithin(b *testing.B) {
	ctx := benchPoisonAccessorCtx(b)
	bench.PoisonAccessorAllocsWithin[*healthChecker](0)(ctx)
}

func BenchmarkPoisonAccessorLatencyWithin(b *testing.B) {
	ctx := benchPoisonAccessorCtx(b)
	bench.PoisonAccessorLatencyWithin[*healthChecker](100 * time.Millisecond)(ctx)
}

func BenchmarkPoisonAccessorConcurrentThroughput(b *testing.B) {
	ctx := benchPoisonAccessorCtx(b)
	bench.PoisonAccessorConcurrentThroughput[*healthChecker](4)(ctx)
}
