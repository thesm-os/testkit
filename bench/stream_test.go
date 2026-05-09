// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package bench_test

import (
	"context"
	"iter"
	"testing"
	"time"

	"go.thesmos.sh/testkit/bench"
	"go.thesmos.sh/testkit/bindings"
)

func benchStreamCtx(b *testing.B, items []string) bench.StreamContext[*listStore, string] {
	b.Helper()
	return bench.StreamContext[*listStore, string]{
		B: b,
		StreamBindings: bindings.StreamBindings[*listStore, string]{
			Factory: func() *listStore { return &listStore{items: items} },
			Call: func(_ context.Context, s *listStore) iter.Seq2[string, error] {
				return s.List()
			},
		},
	}
}

func BenchmarkStreamHotPath(b *testing.B) {
	ctx := benchStreamCtx(b, []string{"a", "b", "c"})
	bench.StreamHotPath[*listStore, string]()(ctx)
}

func BenchmarkStreamAllocsWithin(b *testing.B) {
	ctx := benchStreamCtx(b, []string{"a", "b", "c"})
	// Budget accounts for iterator allocation overhead.
	bench.StreamAllocsWithin[*listStore, string](5)(ctx)
}

func BenchmarkStreamLatencyWithin(b *testing.B) {
	ctx := benchStreamCtx(b, []string{"a", "b", "c"})
	bench.StreamLatencyWithin[*listStore, string](100 * time.Millisecond)(ctx)
}

func BenchmarkStreamConcurrentThroughput(b *testing.B) {
	ctx := benchStreamCtx(b, []string{"a", "b", "c"})
	bench.StreamConcurrentThroughput[*listStore, string](4)(ctx)
}
