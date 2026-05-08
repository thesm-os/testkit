// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package bench_test

import (
	"context"
	"testing"

	"go.thesmos.sh/testkit/bench"
	"go.thesmos.sh/testkit/bindings"
)

func benchLookupCtx(b *testing.B) bench.LookupContext[*lookupStore, string, int64, lookupMeta] {
	b.Helper()
	return bench.LookupContext[*lookupStore, string, int64, lookupMeta]{
		B: b,
		LookupBindings: bindings.LookupBindings[*lookupStore, string, int64, lookupMeta]{
			Factory: newLookupStore,
			Call: func(ctx context.Context, s *lookupStore, k string) (int64, lookupMeta, bool) {
				return s.Inspect(ctx, k)
			},
		},
	}
}

func BenchmarkLookupHotPath(b *testing.B) {
	ctx := benchLookupCtx(b)
	bench.LookupHotPath[*lookupStore, string, int64, lookupMeta]("a")(ctx)
}

func BenchmarkLookupAllocsWithin(b *testing.B) {
	ctx := benchLookupCtx(b)
	bench.LookupAllocsWithin[*lookupStore, string, int64, lookupMeta]("a", 0)(ctx)
}
