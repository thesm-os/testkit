// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package genericstest_test

import (
	"context"
	"testing"

	"go.thesmos.sh/testkit/generator/testdata/generics"
	"go.thesmos.sh/testkit/generator/testdata/generics/genericstest"
)

// BenchmarkKeyMapStub exercises BenchMode against the two-parameter
// generic stub.
func BenchmarkKeyMapStub(b *testing.B) {
	genericstest.BenchmarkKeyMapContract(b, func() generics.KeyMap[string, int] {
		return genericstest.NewKeyMapStub[string, int](b,
			genericstest.KeyMapStubBenchMode[string, int](),
			genericstest.WithKeyMapGet[string, int](func(_ context.Context, _ string) (int, error) {
				return 0, nil
			}),
			genericstest.WithKeyMapSet[string, int](func(_ context.Context, _ string, _ int) error {
				return nil
			}),
		)
	})
}
