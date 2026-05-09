// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package genericstest_test

import (
	"context"
	"testing"

	"go.thesmos.sh/testkit/generator/testdata/generics"
)

// TestHolderContract closes the loop on the generic suite generator.
// AssertHolderContract is parameterized on V, instantiated here at
// V=string. The factory pre-seeds "test-key" → "" so the Reader
// baseline (which expects [Get("test-key")] to return *new(V)) lands
// on the seeded zero value rather than the in-mem's ErrNotFound miss.
func TestHolderContract(t *testing.T) {
	t.Parallel()
	AssertHolderContract(t, func() generics.Holder[string] {
		h := generics.NewInMemoryHolder[string]()
		_ = h.Put(context.Background(), "test-key", "")
		return h
	})
}
