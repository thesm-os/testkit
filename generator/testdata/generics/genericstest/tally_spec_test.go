// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package genericstest_test

import (
	"testing"

	"go.thesmos.sh/testkit/generator/testdata/generics"
)

// TestTallyContract closes the loop on the constrained-T generic
// suite generator. AssertTallyContract instantiates at T=int.
// No seeding needed: AssertCompositeWriteSucceeds calls Add(0) and
// AssertAggregatorReturns expects Total() to return the zero T,
// which an empty in-mem already produces.
func TestTallyContract(t *testing.T) {
	t.Parallel()
	AssertTallyContract(t, func() generics.Tally[int] {
		return generics.NewInMemoryTally[int]()
	})
}
