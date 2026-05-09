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

func benchVoidLifecycleCtx(b *testing.B) bench.VoidLifecycleContext[*resetter] {
	b.Helper()
	return bench.VoidLifecycleContext[*resetter]{
		B: b,
		VoidLifecycleBindings: bindings.VoidLifecycleBindings[*resetter]{
			Factory: newResetter,
			Call: func(ctx context.Context, r *resetter) {
				r.Reset(ctx)
			},
		},
	}
}

func BenchmarkVoidLifecycleHotPath(b *testing.B) {
	ctx := benchVoidLifecycleCtx(b)
	bench.VoidLifecycleHotPath[*resetter]()(ctx)
}

func BenchmarkVoidLifecycleAllocsWithin(b *testing.B) {
	ctx := benchVoidLifecycleCtx(b)
	bench.VoidLifecycleAllocsWithin[*resetter](0)(ctx)
}

func BenchmarkVoidLifecycleLatencyWithin(b *testing.B) {
	ctx := benchVoidLifecycleCtx(b)
	bench.VoidLifecycleLatencyWithin[*resetter](100 * time.Millisecond)(ctx)
}

func BenchmarkVoidLifecycleConcurrentThroughput(b *testing.B) {
	ctx := benchVoidLifecycleCtx(b)
	bench.VoidLifecycleConcurrentThroughput[*resetter](4)(ctx)
}
