// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package bench_test

import (
	"context"
	"testing"

	"go.thesmos.sh/testkit/bench"
	"go.thesmos.sh/testkit/bindings"
)

func benchReaderCtx(b *testing.B, data map[string]string) bench.ReaderContext[*mapReader, string, string] {
	b.Helper()
	return bench.ReaderContext[*mapReader, string, string]{
		B: b,
		ReaderBindings: bindings.ReaderBindings[*mapReader, string, string]{
			Factory: func() *mapReader { return newMapReader(data) },
			Call: func(ctx context.Context, r *mapReader, k string) (string, error) {
				return r.Get(ctx, k)
			},
		},
	}
}

func BenchmarkReaderHotPath(b *testing.B) {
	ctx := benchReaderCtx(b, map[string]string{"a": "alpha"})
	bench.ReaderHotPath[*mapReader, string, string]("a")(ctx)
}

func BenchmarkReaderAllocsWithin(b *testing.B) {
	ctx := benchReaderCtx(b, map[string]string{"a": "alpha"})
	bench.ReaderAllocsWithin[*mapReader, string, string]("a", 0)(ctx)
}

func BenchmarkReaderConcurrentThroughput(b *testing.B) {
	ctx := benchReaderCtx(b, map[string]string{"a": "alpha"})
	bench.ReaderConcurrentThroughput[*mapReader, string, string]("a", 4)(ctx)
}
