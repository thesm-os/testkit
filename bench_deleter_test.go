// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package testkit_test

import (
	"context"
	"testing"

	"go.thesmos.sh/testkit"
)

func benchDeleterCtx(b *testing.B) testkit.BenchDeleterContext[*delStore, string] {
	b.Helper()
	return testkit.BenchDeleterContext[*delStore, string]{
		B: b,
		DeleterBindings: testkit.DeleterBindings[*delStore, string]{
			Factory: newDelStore,
			Call: func(ctx context.Context, s *delStore, k string) error {
				return s.Delete(ctx, k)
			},
		},
	}
}

func BenchmarkDeleterHotPath(b *testing.B) {
	ctx := benchDeleterCtx(b)
	testkit.BenchDeleterHotPath[*delStore, string]("existing")(ctx)
}

func BenchmarkDeleterAllocsWithin(b *testing.B) {
	ctx := benchDeleterCtx(b)
	// Budget accounts for factory allocation inside AllocsPerRun closure.
	testkit.BenchDeleterAllocsWithin[*delStore, string]("existing", 5)(ctx)
}
