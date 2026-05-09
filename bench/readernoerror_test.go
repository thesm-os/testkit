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

func benchReaderNoErrorCtx(b *testing.B) bench.ReaderNoErrorContext[*infallibleStore, string, string] {
	b.Helper()
	return bench.ReaderNoErrorContext[*infallibleStore, string, string]{
		B: b,
		ReaderNoErrorBindings: bindings.ReaderNoErrorBindings[*infallibleStore, string, string]{
			Factory: func() *infallibleStore { return newInfallibleStore(map[string]string{"a": "alpha"}) },
			Call: func(ctx context.Context, s *infallibleStore, k string) string {
				return s.Get(ctx, k)
			},
		},
	}
}

func BenchmarkReaderNoErrorHotPath(b *testing.B) {
	ctx := benchReaderNoErrorCtx(b)
	bench.ReaderNoErrorHotPath[*infallibleStore, string, string]("a")(ctx)
}

func BenchmarkReaderNoErrorAllocsWithin(b *testing.B) {
	ctx := benchReaderNoErrorCtx(b)
	bench.ReaderNoErrorAllocsWithin[*infallibleStore, string, string]("a", 0)(ctx)
}

func BenchmarkReaderNoErrorLatencyWithin(b *testing.B) {
	ctx := benchReaderNoErrorCtx(b)
	bench.ReaderNoErrorLatencyWithin[*infallibleStore, string, string]("a", 100*time.Millisecond)(ctx)
}

func BenchmarkReaderNoErrorConcurrentThroughput(b *testing.B) {
	ctx := benchReaderNoErrorCtx(b)
	bench.ReaderNoErrorConcurrentThroughput[*infallibleStore, string, string]("a", 4)(ctx)
}
