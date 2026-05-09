// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package genericstest_test

import (
	"testing"

	"go.thesmos.sh/testkit/generator/testdata/generics"
	"go.thesmos.sh/testkit/generator/testdata/generics/genericstest"
)

// BenchmarkTally closes the loop on `testkit bench` for the
// constrained-T generic. T=int satisfies the Numeric constraint;
// no seeding is needed since the always-emit Aggregator/CompositeWriter
// primitives operate on freshly produced impls.
func BenchmarkTally(b *testing.B) {
	genericstest.BenchmarkTallyContract(b, func() generics.Tally[int] {
		return generics.NewInMemoryTally[int]()
	})
}
