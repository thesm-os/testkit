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

func benchCompositeWriterCtx(b *testing.B) bench.CompositeWriterContext[*nsStore, string, string] {
	b.Helper()
	return bench.CompositeWriterContext[*nsStore, string, string]{
		B: b,
		CompositeWriterBindings: bindings.CompositeWriterBindings[*nsStore, string, string]{
			Factory: newNsStore,
			Call: func(ctx context.Context, s *nsStore, ns, value string) error {
				return s.Put(ctx, ns, value)
			},
		},
	}
}

func BenchmarkCompositeWriterHotPath(b *testing.B) {
	ctx := benchCompositeWriterCtx(b)
	bench.CompositeWriterHotPath[*nsStore, string, string]("ns", "value")(ctx)
}

func BenchmarkCompositeWriterAllocsWithin(b *testing.B) {
	ctx := benchCompositeWriterCtx(b)
	// Append-into-slice allocates as the slice grows; budget reflects amortized growth.
	bench.CompositeWriterAllocsWithin[*nsStore, string, string]("ns", "value", 1)(ctx)
}

func BenchmarkCompositeWriterLatencyWithin(b *testing.B) {
	ctx := benchCompositeWriterCtx(b)
	bench.CompositeWriterLatencyWithin[*nsStore, string, string]("ns", "value", 100*time.Millisecond)(ctx)
}

func BenchmarkCompositeWriterConcurrentThroughput(b *testing.B) {
	ctx := benchCompositeWriterCtx(b)
	bench.CompositeWriterConcurrentThroughput[*nsStore, string, string]("ns", "value", 4)(ctx)
}
