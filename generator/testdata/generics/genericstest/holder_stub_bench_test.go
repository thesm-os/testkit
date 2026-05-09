// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package genericstest_test

import (
	"context"
	"testing"

	"go.thesmos.sh/testkit/generator/testdata/generics"
	"go.thesmos.sh/testkit/generator/testdata/generics/genericstest"
)

// BenchmarkHolderStub runs the contract driver against the generated
// stub in BenchMode. With recording disabled and Func overrides
// installed for every method, the stub dispatch must be alloc-free
// — bridging the stub's "transparent recording" claim and the bench
// infrastructure on a single-parameter generic.
func BenchmarkHolderStub(b *testing.B) {
	genericstest.BenchmarkHolderContract(b, func() generics.Holder[string] {
		return genericstest.NewHolderStub[string](b,
			genericstest.HolderStubBenchMode[string](),
			genericstest.WithHolderGet[string](func(_ context.Context, _ string) (string, error) {
				return "", nil
			}),
			genericstest.WithHolderPut[string](func(_ context.Context, _ string, _ string) error {
				return nil
			}),
			genericstest.WithHolderDelete[string](func(_ context.Context, _ string) error {
				return nil
			}),
		)
	})
}
