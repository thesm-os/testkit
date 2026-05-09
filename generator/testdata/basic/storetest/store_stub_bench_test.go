// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package storetest_test

import (
	"context"
	"testing"

	"go.thesmos.sh/testkit/generator/testdata/basic"
	"go.thesmos.sh/testkit/generator/testdata/basic/storetest"
)

// BenchmarkStoreStub runs the contract driver against the generated
// stub in BenchMode (recording disabled, fault chain disabled). The
// stub is the recording test-double; with BenchMode active, it must
// be alloc-free on the dispatch path — the bench output proves it.
//
// This bridges the stub generator's "transparent recording" claim
// (G16 in ANALYSIS.md) and the bench infrastructure: instead of
// asserting alloc-free on a hand-written impl, we assert it on the
// generated stub. Any future change to the stub dispatch chain that
// reintroduces an allocation surfaces here.
func BenchmarkStoreStub(b *testing.B) {
	storetest.BenchmarkStoreContract(b, func() basic.Store {
		s := storetest.NewStoreStub(b, storetest.StoreStubBenchMode())
		s.OnGet.Func(func(_ context.Context, _ string) (basic.Item, error) {
			return basic.Item{ID: "test-id"}, nil
		})
		s.OnPut.Func(func(_ context.Context, _ basic.Item) error {
			return nil
		})
		return s
	})
}
