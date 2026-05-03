// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package bench_test

import (
	"context"
	"testing"

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

func BenchmarkLifecycleAllocsWithin(b *testing.B) {
	ctx := benchLifecycleCtx(b)
	bench.LifecycleAllocsWithin[*lifecycle](0)(ctx)
}
