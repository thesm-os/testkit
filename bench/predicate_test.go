// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package bench_test

import (
	"testing"

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

func BenchmarkPredicateAllocsWithin(b *testing.B) {
	ctx := benchPredicateCtx(b)
	bench.PredicateAllocsWithin[*validator](0)(ctx)
}
