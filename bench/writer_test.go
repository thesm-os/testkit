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

func benchWriterCtx(b *testing.B) bench.WriterContext[*mapWriter, entry] {
	b.Helper()
	return bench.WriterContext[*mapWriter, entry]{
		B: b,
		WriterBindings: bindings.WriterBindings[*mapWriter, entry]{
			Factory: newMapWriter,
			Call: func(ctx context.Context, w *mapWriter, e entry) error {
				return w.Put(ctx, e)
			},
		},
	}
}

func BenchmarkWriterHotPath(b *testing.B) {
	ctx := benchWriterCtx(b)
	bench.WriterHotPath[*mapWriter, entry](entry{Key: "a", Value: "alpha"})(ctx)
}

func BenchmarkWriterAllocsWithin(b *testing.B) {
	ctx := benchWriterCtx(b)
	bench.WriterAllocsWithin[*mapWriter, entry](entry{Key: "a", Value: "alpha"}, 1)(ctx)
}

func BenchmarkWriterLatencyWithin(b *testing.B) {
	ctx := benchWriterCtx(b)
	bench.WriterLatencyWithin[*mapWriter, entry](entry{Key: "a", Value: "alpha"}, 100*time.Millisecond)(ctx)
}

func BenchmarkWriterConcurrentThroughput(b *testing.B) {
	ctx := benchWriterCtx(b)
	bench.WriterConcurrentThroughput[*mapWriter, entry](entry{Key: "a", Value: "alpha"}, 4)(ctx)
}
