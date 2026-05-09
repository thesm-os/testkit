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

func benchReaderWithBoolCtx(b *testing.B) bench.ReaderWithBoolContext[*boolMap, string, int64] {
	b.Helper()
	return bench.ReaderWithBoolContext[*boolMap, string, int64]{
		B: b,
		ReaderWithBoolBindings: bindings.ReaderWithBoolBindings[*boolMap, string, int64]{
			Factory: func() *boolMap { return newBoolMap(map[string]int64{"a": 10}) },
			Call: func(ctx context.Context, m *boolMap, k string) (int64, bool) {
				return m.Load(ctx, k)
			},
		},
	}
}

func BenchmarkReaderWithBoolHotPath(b *testing.B) {
	ctx := benchReaderWithBoolCtx(b)
	bench.ReaderWithBoolHotPath[*boolMap, string, int64]("a")(ctx)
}

func BenchmarkReaderWithBoolAllocsWithin(b *testing.B) {
	ctx := benchReaderWithBoolCtx(b)
	bench.ReaderWithBoolAllocsWithin[*boolMap, string, int64]("a", 0)(ctx)
}

func BenchmarkReaderWithBoolLatencyWithin(b *testing.B) {
	ctx := benchReaderWithBoolCtx(b)
	bench.ReaderWithBoolLatencyWithin[*boolMap, string, int64]("a", 100*time.Millisecond)(ctx)
}

func BenchmarkReaderWithBoolConcurrentThroughput(b *testing.B) {
	ctx := benchReaderWithBoolCtx(b)
	bench.ReaderWithBoolConcurrentThroughput[*boolMap, string, int64]("a", 4)(ctx)
}
