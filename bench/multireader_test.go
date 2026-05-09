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

func benchMultiReaderCtx(b *testing.B) bench.MultiReaderContext[*metaStore, string, string, int] {
	b.Helper()
	return bench.MultiReaderContext[*metaStore, string, string, int]{
		B: b,
		MultiReaderBindings: bindings.MultiReaderBindings[*metaStore, string, string, int]{
			Factory: newMetaStore,
			Call: func(ctx context.Context, s *metaStore, k string) (string, int, error) {
				return s.Inspect(ctx, k)
			},
		},
	}
}

func BenchmarkMultiReaderHotPath(b *testing.B) {
	ctx := benchMultiReaderCtx(b)
	bench.MultiReaderHotPath[*metaStore, string, string, int]("a")(ctx)
}

func BenchmarkMultiReaderAllocsWithin(b *testing.B) {
	ctx := benchMultiReaderCtx(b)
	bench.MultiReaderAllocsWithin[*metaStore, string, string, int]("a", 0)(ctx)
}

func BenchmarkMultiReaderLatencyWithin(b *testing.B) {
	ctx := benchMultiReaderCtx(b)
	bench.MultiReaderLatencyWithin[*metaStore, string, string, int]("a", 100*time.Millisecond)(ctx)
}

func BenchmarkMultiReaderConcurrentThroughput(b *testing.B) {
	ctx := benchMultiReaderCtx(b)
	bench.MultiReaderConcurrentThroughput[*metaStore, string, string, int]("a", 4)(ctx)
}
