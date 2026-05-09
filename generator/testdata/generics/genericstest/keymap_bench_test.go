// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package genericstest_test

import (
	"context"
	"testing"

	"go.thesmos.sh/testkit/generator/testdata/generics"
	"go.thesmos.sh/testkit/generator/testdata/generics/genericstest"
)

// BenchmarkKeyMap closes the loop on `testkit bench` for the
// two-parameter generic. The factory pre-seeds the zero (K, V) pair
// so generic Reader primitives whose sample key renders as `*new(K)`
// land on a populated entry rather than a miss.
func BenchmarkKeyMap(b *testing.B) {
	genericstest.BenchmarkKeyMapContract(b, func() generics.KeyMap[string, int] {
		m := generics.NewInMemoryKeyMap[string, int]()
		_ = m.Set(context.Background(), "", 0)
		return m
	})
}
