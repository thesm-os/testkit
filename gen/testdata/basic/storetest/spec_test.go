// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package storetest_test

import (
	"context"
	"testing"

	"go.thesmos.sh/testkit/bench"
	"go.thesmos.sh/testkit/gen/testdata/basic"
	"go.thesmos.sh/testkit/gen/testdata/basic/storetest"
	"go.thesmos.sh/testkit/suite"
)

// --- Factories ---

func inmemoryFactory() basic.Store {
	s := basic.NewInMemoryStore()
	_ = s.Put(context.Background(), basic.Item{ID: "seed", Name: "seed"})
	return s
}

func stubFactory() basic.Store {
	return storetest.NewStoreStub(nil, storetest.StoreStubDelegateTo(inmemoryFactory()))
}

func stubBenchFactory() basic.Store {
	return storetest.NewStoreStub(nil,
		storetest.StoreStubBenchMode(),
		storetest.StoreStubDelegateTo(inmemoryFactory()))
}

// --- Suite options shared across impls ---

func suiteOpts() []storetest.StoreOption {
	return []storetest.StoreOption{
		storetest.StorePrePopulate(func(ctx context.Context, s basic.Store) {
			_ = s.Put(ctx, basic.Item{ID: "pre", Name: "prepopulated"})
		}),
		storetest.StoreOnGet(
			suite.AssertReturnsSentinel[basic.Store, string, basic.Item](
				"nonexistent", basic.ErrNotFound,
			),
		),
		storetest.StoreOnDelete(
			suite.AssertDeleteSucceeds[basic.Store, string]("seed"),
		),
		storetest.StoreOnPing(
			suite.AssertLifecycleSucceeds[basic.Store](),
		),
	}
}

func benchOpts() []storetest.StoreBenchOption {
	return []storetest.StoreBenchOption{
		storetest.StoreBenchOnGet(
			bench.ReaderHotPath[basic.Store, string, basic.Item]("seed"),
		),
		storetest.StoreBenchOnPing(
			bench.LifecycleAllocsWithin[basic.Store](0),
		),
	}
}

// --- InMemory impl ---

func TestStoreContract_InMemory(t *testing.T) {
	t.Parallel()
	storetest.AssertStoreContract(t, inmemoryFactory, suiteOpts()...)
}

func BenchmarkStoreContract_InMemory(b *testing.B) {
	storetest.BenchmarkStoreContract(b, inmemoryFactory, benchOpts()...)
}

// --- Stub+DelegateTo impl (same contract, different impl) ---

func TestStoreContract_StubDelegateTo(t *testing.T) {
	t.Parallel()
	storetest.AssertStoreContract(t, stubFactory, suiteOpts()...)
}

func BenchmarkStoreContract_StubDelegateTo(b *testing.B) {
	storetest.BenchmarkStoreContract(b, stubFactory, benchOpts()...)
}

// --- Stub+BenchMode (zero-alloc hot path) ---

func BenchmarkStoreContract_StubBenchMode(b *testing.B) {
	storetest.BenchmarkStoreContract(b, stubBenchFactory, benchOpts()...)
}
