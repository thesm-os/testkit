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

func benchPointerReaderCtx(b *testing.B) bench.PointerReaderContext[*ptrStore, string, string] {
	b.Helper()
	return bench.PointerReaderContext[*ptrStore, string, string]{
		B: b,
		PointerReaderBindings: bindings.PointerReaderBindings[*ptrStore, string, string]{
			Factory: newPtrStore,
			Call: func(ctx context.Context, s *ptrStore, k string) *string {
				return s.Find(ctx, k)
			},
		},
	}
}

func BenchmarkPointerReaderHotPath(b *testing.B) {
	ctx := benchPointerReaderCtx(b)
	bench.PointerReaderHotPath[*ptrStore, string, string]("a")(ctx)
}

func BenchmarkPointerReaderAllocsWithin(b *testing.B) {
	ctx := benchPointerReaderCtx(b)
	bench.PointerReaderAllocsWithin[*ptrStore, string, string]("a", 0)(ctx)
}

func BenchmarkPointerReaderLatencyWithin(b *testing.B) {
	ctx := benchPointerReaderCtx(b)
	bench.PointerReaderLatencyWithin[*ptrStore, string, string]("a", 100*time.Millisecond)(ctx)
}

func BenchmarkPointerReaderConcurrentThroughput(b *testing.B) {
	ctx := benchPointerReaderCtx(b)
	bench.PointerReaderConcurrentThroughput[*ptrStore, string, string]("a", 4)(ctx)
}
