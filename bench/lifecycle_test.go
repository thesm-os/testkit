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

func benchLifecycleCtx(b *testing.B) bench.LifecycleContext[*lifecycle] {
	b.Helper()
	return bench.LifecycleContext[*lifecycle]{
		B: b,
		LifecycleBindings: bindings.LifecycleBindings[*lifecycle]{
			Factory: newLifecycle,
			Call: func(ctx context.Context, l *lifecycle) error {
				return l.Open(ctx)
			},
		},
	}
}

func BenchmarkLifecycleHotPath(b *testing.B) {
	ctx := benchLifecycleCtx(b)
	bench.LifecycleHotPath[*lifecycle]()(ctx)
}

func BenchmarkLifecycleAllocsWithin(b *testing.B) {
	ctx := benchLifecycleCtx(b)
	bench.LifecycleAllocsWithin[*lifecycle](0)(ctx)
}

func BenchmarkLifecycleLatencyWithin(b *testing.B) {
	ctx := benchLifecycleCtx(b)
	bench.LifecycleLatencyWithin[*lifecycle](100 * time.Millisecond)(ctx)
}

func BenchmarkLifecycleConcurrentThroughput(b *testing.B) {
	ctx := benchLifecycleCtx(b)
	bench.LifecycleConcurrentThroughput[*lifecycle](4)(ctx)
}
