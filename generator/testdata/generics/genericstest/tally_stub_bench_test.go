// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package genericstest_test

import (
	"context"
	"testing"

	"go.thesmos.sh/testkit/generator/testdata/generics"
	"go.thesmos.sh/testkit/generator/testdata/generics/genericstest"
)

// BenchmarkTallyStub exercises BenchMode against the constrained-T
// generic stub.
func BenchmarkTallyStub(b *testing.B) {
	genericstest.BenchmarkTallyContract(b, func() generics.Tally[int] {
		return genericstest.NewTallyStub[int](b,
			genericstest.TallyStubBenchMode[int](),
			genericstest.WithTallyAdd[int](func(_ context.Context, _ string, _ int) error {
				return nil
			}),
			genericstest.WithTallyTotal[int](func(_ context.Context) (int, error) {
				return 0, nil
			}),
		)
	})
}
