// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package bench_test

import (
	"testing"
	"time"

	"go.thesmos.sh/testkit/bench"
	"go.thesmos.sh/testkit/bindings"
)

func benchPredicateCtx(b *testing.B) bench.PredicateContext[*validator] {
	b.Helper()
	return bench.PredicateContext[*validator]{
		B: b,
		PredicateBindings: bindings.PredicateBindings[*validator]{
			Factory: func() *validator { return newValidator(true) },
			Call:    func(v *validator) bool { return v.IsValid() },
		},
	}
}

func BenchmarkPredicateHotPath(b *testing.B) {
	ctx := benchPredicateCtx(b)
	bench.PredicateHotPath[*validator]()(ctx)
}

func BenchmarkPredicateAllocsWithin(b *testing.B) {
	ctx := benchPredicateCtx(b)
	bench.PredicateAllocsWithin[*validator](0)(ctx)
}

func BenchmarkPredicateLatencyWithin(b *testing.B) {
	ctx := benchPredicateCtx(b)
	bench.PredicateLatencyWithin[*validator](100 * time.Millisecond)(ctx)
}

func BenchmarkPredicateConcurrentThroughput(b *testing.B) {
	ctx := benchPredicateCtx(b)
	bench.PredicateConcurrentThroughput[*validator](4)(ctx)
}
