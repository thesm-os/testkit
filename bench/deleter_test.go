// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package bench_test

import (
	"context"
	"testing"

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
	// Budget accounts for factory allocation inside AllocsPerRun closure.
	bench.DeleterAllocsWithin[*delStore, string]("existing", 5)(ctx)
}
