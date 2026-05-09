// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package genericstest_test

import (
	"context"
	"testing"

	"go.thesmos.sh/testkit/generator/testdata/generics"
	"go.thesmos.sh/testkit/generator/testdata/generics/genericstest"
)

// BenchmarkHolder closes the loop on `testkit bench` for the
// single-parameter generic. The factory pre-seeds "test-key" → "" so
// every always-emit primitive observes a real entry — a contract-
// correct sample would otherwise hit the miss path on every iteration
// and report skewed allocation counts.
func BenchmarkHolder(b *testing.B) {
	genericstest.BenchmarkHolderContract(b, func() generics.Holder[string] {
		h := generics.NewInMemoryHolder[string]()
		_ = h.Put(context.Background(), "test-key", "")
		return h
	})
}
