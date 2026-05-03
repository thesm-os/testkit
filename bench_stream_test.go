// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package testkit_test

import (
	"context"
	"iter"
	"testing"

	"go.thesmos.sh/testkit"
)

func benchStreamCtx(b *testing.B, items []string) testkit.BenchStreamContext[*listStore, string] {
	b.Helper()
	return testkit.BenchStreamContext[*listStore, string]{
		B: b,
		StreamBindings: testkit.StreamBindings[*listStore, string]{
			Factory: func() *listStore { return &listStore{items: items} },
			Call: func(_ context.Context, s *listStore) iter.Seq2[string, error] {
				return s.List()
			},
		},
	}
}

func BenchmarkStreamHotPath(b *testing.B) {
	ctx := benchStreamCtx(b, []string{"a", "b", "c"})
	testkit.BenchStreamHotPath[*listStore, string]()(ctx)
}

func BenchmarkStreamAllocsWithin(b *testing.B) {
	ctx := benchStreamCtx(b, []string{"a", "b", "c"})
	// Budget accounts for iterator allocation overhead.
	testkit.BenchStreamAllocsWithin[*listStore, string](5)(ctx)
}
