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

func benchBatchReaderCtx(b *testing.B, data map[string]string) bench.BatchReaderContext[*batchStore, string, string] {
	b.Helper()
	return bench.BatchReaderContext[*batchStore, string, string]{
		B: b,
		BatchReaderBindings: bindings.BatchReaderBindings[*batchStore, string, string]{
			Factory: func() *batchStore { return newBatchStore(data) },
			Call: func(ctx context.Context, s *batchStore, keys []string) ([]string, error) {
				return s.GetMany(ctx, keys)
			},
		},
	}
}

func BenchmarkBatchReaderHotPath(b *testing.B) {
	ctx := benchBatchReaderCtx(b, map[string]string{"a": "alpha", "b": "beta", "c": "gamma"})
	bench.BatchReaderHotPath[*batchStore, string, string]([]string{"a", "b", "c"})(ctx)
}

func BenchmarkBatchReaderAllocsWithin(b *testing.B) {
	ctx := benchBatchReaderCtx(b, map[string]string{"a": "alpha", "b": "beta", "c": "gamma"})
	// Budget accounts for output slice allocation.
	bench.BatchReaderAllocsWithin[*batchStore, string, string]([]string{"a", "b", "c"}, 1)(ctx)
}

func BenchmarkBatchReaderLatencyWithin(b *testing.B) {
	ctx := benchBatchReaderCtx(b, map[string]string{"a": "alpha", "b": "beta", "c": "gamma"})
	bench.BatchReaderLatencyWithin[*batchStore, string, string]([]string{"a", "b", "c"}, 100*time.Millisecond)(ctx)
}

func BenchmarkBatchReaderConcurrentThroughput(b *testing.B) {
	ctx := benchBatchReaderCtx(b, map[string]string{"a": "alpha", "b": "beta", "c": "gamma"})
	bench.BatchReaderConcurrentThroughput[*batchStore, string, string]([]string{"a", "b", "c"}, 4)(ctx)
}
