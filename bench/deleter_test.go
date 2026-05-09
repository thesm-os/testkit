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

func benchDeleterCtx(b *testing.B) bench.DeleterContext[*delStore, string] {
	b.Helper()
	return bench.DeleterContext[*delStore, string]{
		B: b,
		DeleterBindings: bindings.DeleterBindings[*delStore, string]{
			Factory: newDelStore,
			Call: func(ctx context.Context, s *delStore, k string) error {
				return s.Delete(ctx, k)
			},
		},
	}
}

func BenchmarkDeleterHotPath(b *testing.B) {
	ctx := benchDeleterCtx(b)
	bench.DeleterHotPath[*delStore, string]("existing")(ctx)
}

func BenchmarkDeleterAllocsWithin(b *testing.B) {
	ctx := benchDeleterCtx(b)
	// First iteration deletes; subsequent iterations hit the not-found path
	// that constructs the sentinel error.
	bench.DeleterAllocsWithin[*delStore, string]("existing", 1)(ctx)
}

func BenchmarkDeleterLatencyWithin(b *testing.B) {
	ctx := benchDeleterCtx(b)
	bench.DeleterLatencyWithin[*delStore, string]("existing", 100*time.Millisecond)(ctx)
}

func BenchmarkDeleterConcurrentThroughput(b *testing.B) {
	ctx := benchDeleterCtx(b)
	bench.DeleterConcurrentThroughput[*delStore, string]("existing", 4)(ctx)
}
