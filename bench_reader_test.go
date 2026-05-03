// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package testkit_test

import (
	"context"
	"testing"

	"go.thesmos.sh/testkit"
)

func benchReaderCtx(b *testing.B, data map[string]string) testkit.BenchReaderContext[*mapReader, string, string] {
	b.Helper()
	return testkit.BenchReaderContext[*mapReader, string, string]{
		B: b,
		ReaderBindings: testkit.ReaderBindings[*mapReader, string, string]{
			Factory: func() *mapReader { return newMapReader(data) },
			Call: func(ctx context.Context, r *mapReader, k string) (string, error) {
				return r.Get(ctx, k)
			},
		},
	}
}

func BenchmarkReaderHotPath(b *testing.B) {
	ctx := benchReaderCtx(b, map[string]string{"a": "alpha"})
	testkit.BenchReaderHotPath[*mapReader, string, string]("a")(ctx)
}

func BenchmarkReaderAllocsWithin(b *testing.B) {
	ctx := benchReaderCtx(b, map[string]string{"a": "alpha"})
	testkit.BenchReaderAllocsWithin[*mapReader, string, string]("a", 0)(ctx)
}

func BenchmarkReaderConcurrentThroughput(b *testing.B) {
	ctx := benchReaderCtx(b, map[string]string{"a": "alpha"})
	testkit.BenchReaderConcurrentThroughput[*mapReader, string, string]("a", 4)(ctx)
}
