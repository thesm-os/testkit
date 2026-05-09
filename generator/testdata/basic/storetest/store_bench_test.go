// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package storetest_test

import (
	"testing"

	"go.thesmos.sh/testkit/generator/testdata/basic"
	"go.thesmos.sh/testkit/generator/testdata/basic/storetest"
)

// BenchmarkStore closes the loop on `testkit bench`: the generated
// [storetest.BenchmarkStoreContract] driver runs against the same
// in-mem impl the suite contract exercises, proving the rendered
// template links against the bench runtime with the right type
// args.
//
// The factory pre-seeds the same key/value the always-emitted
// hot-path benchmark targets so per-method primitives observe a
// contract-correct impl.
func BenchmarkStore(b *testing.B) {
	storetest.BenchmarkStoreContract(b, func() basic.Store {
		s := basic.NewInMemoryStore()
		s.Seed("test-key", basic.Item{ID: "test-id"})
		return s
	})
}
