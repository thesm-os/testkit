// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package bench_test

import (
	"context"
	"testing"

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
	// Budget accounts for factory allocation inside AllocsPerRun closure.
	bench.WriterAllocsWithin[*mapWriter, entry](entry{Key: "a", Value: "alpha"}, 5)(ctx)
}
