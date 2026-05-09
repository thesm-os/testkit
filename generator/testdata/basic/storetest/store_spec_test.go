// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package storetest_test

import (
	"testing"

	"go.thesmos.sh/testkit/generator/testdata/basic"
)

// TestStoreContract closes the loop on `testkit suite`: the
// generated [AssertStoreContract] driver runs against a real impl,
// proving the rendered template links against the suite runtime
// with the right type args and that the per-shape baselines
// actually pass when the impl is contract-correct.
//
// The factory pre-seeds "test-key" → Item{ID: "test-id"} because
// the generated baseline asserts Get("test-key") == Item{ID:
// "test-id"} — sample-driven literals are part of the contract.
func TestStoreContract(t *testing.T) {
	t.Parallel()
	AssertStoreContract(t, func() basic.Store {
		s := basic.NewInMemoryStore()
		s.Seed("test-key", basic.Item{ID: "test-id"})
		return s
	})
}
