// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package bench_test

import (
	"testing"
	"time"

	"go.thesmos.sh/testkit/bench"
	"go.thesmos.sh/testkit/bindings"
)

func benchPureCtx(b *testing.B) bench.PureContext[*counter, int] {
	b.Helper()
	return bench.PureContext[*counter, int]{
		B: b,
		PureBindings: bindings.PureBindings[*counter, int]{
			Factory: newCounter,
			Call:    func(c *counter) int { return c.Value() },
		},
	}
}

func BenchmarkPureHotPath(b *testing.B) {
	ctx := benchPureCtx(b)
	bench.PureHotPath[*counter, int]()(ctx)
}

func BenchmarkPureAllocsWithin(b *testing.B) {
	ctx := benchPureCtx(b)
	bench.PureAllocsWithin[*counter, int](0)(ctx)
}

func BenchmarkPureLatencyWithin(b *testing.B) {
	ctx := benchPureCtx(b)
	bench.PureLatencyWithin[*counter, int](100 * time.Millisecond)(ctx)
}

func BenchmarkPureConcurrentThroughput(b *testing.B) {
	ctx := benchPureCtx(b)
	bench.PureConcurrentThroughput[*counter, int](4)(ctx)
}
