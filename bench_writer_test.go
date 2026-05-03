// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package testkit_test

import (
	"context"
	"testing"

	"go.thesmos.sh/testkit"
)

func benchWriterCtx(b *testing.B) testkit.BenchWriterContext[*mapWriter, entry] {
	b.Helper()
	return testkit.BenchWriterContext[*mapWriter, entry]{
		B: b,
		WriterBindings: testkit.WriterBindings[*mapWriter, entry]{
			Factory: newMapWriter,
			Call: func(ctx context.Context, w *mapWriter, e entry) error {
				return w.Put(ctx, e)
			},
		},
	}
}

func BenchmarkWriterHotPath(b *testing.B) {
	ctx := benchWriterCtx(b)
	testkit.BenchWriterHotPath[*mapWriter, entry](entry{Key: "a", Value: "alpha"})(ctx)
}

func BenchmarkWriterAllocsWithin(b *testing.B) {
	ctx := benchWriterCtx(b)
	// Budget accounts for factory allocation inside AllocsPerRun closure.
	testkit.BenchWriterAllocsWithin[*mapWriter, entry](entry{Key: "a", Value: "alpha"}, 5)(ctx)
}
