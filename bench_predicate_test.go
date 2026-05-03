// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package testkit_test

import (
	"testing"

	"go.thesmos.sh/testkit"
)

func benchPredicateCtx(b *testing.B) testkit.BenchPredicateContext[*validator] {
	b.Helper()
	return testkit.BenchPredicateContext[*validator]{
		B: b,
		PredicateBindings: testkit.PredicateBindings[*validator]{
			Factory: func() *validator { return newValidator(true) },
			Call:    func(v *validator) bool { return v.IsValid() },
		},
	}
}

func BenchmarkPredicateAllocsWithin(b *testing.B) {
	ctx := benchPredicateCtx(b)
	testkit.BenchPredicateAllocsWithin[*validator](0)(ctx)
}
