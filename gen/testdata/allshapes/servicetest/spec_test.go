// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package servicetest_test

import (
	"context"
	"testing"

	"go.thesmos.sh/testkit/bench"
	"go.thesmos.sh/testkit/gen/testdata/allshapes"
	"go.thesmos.sh/testkit/gen/testdata/allshapes/servicetest"
	"go.thesmos.sh/testkit/suite"
)

func inmemoryFactory() allshapes.Service {
	s := allshapes.NewInMemoryService()
	_ = s.Put(context.Background(), allshapes.Item{ID: "seed", Name: "seed"})
	return s
}

func stubFactory() allshapes.Service {
	return servicetest.NewServiceStub(nil,
		servicetest.ServiceStubDelegateTo(inmemoryFactory()))
}

func stubBenchFactory() allshapes.Service {
	return servicetest.NewServiceStub(nil,
		servicetest.ServiceStubBenchMode(),
		servicetest.ServiceStubDelegateTo(inmemoryFactory()))
}

func suiteOpts() []servicetest.ServiceOption {
	return []servicetest.ServiceOption{
		servicetest.ServiceOnGet(
			suite.AssertReturnsSentinel[allshapes.Service, string, allshapes.Item](
				"nonexistent", allshapes.ErrNotFound),
		),
		servicetest.ServiceOnDelete(
			suite.AssertDeleteSucceeds[allshapes.Service, string]("seed"),
		),
		servicetest.ServiceOnCount(
			suite.AssertAggregatorConsistent[allshapes.Service, int](3),
		),
		servicetest.ServiceOnClose(
			suite.AssertLifecycleSucceeds[allshapes.Service](),
		),
		servicetest.ServiceOnDescribe(
			suite.AssertDeterministic[allshapes.Service, string](3),
		),
		servicetest.ServiceOnIsEmpty(
			suite.AssertPredicateConsistent[allshapes.Service](3),
		),
		servicetest.ServiceOnList(
			suite.AssertStreamCompletes[allshapes.Service, allshapes.Item](),
		),
		servicetest.ServiceOnTouch(
			suite.AssertMutatorSucceeds[allshapes.Service, string]("seed"),
		),
		servicetest.ServiceOnLoad(
			suite.AssertReaderWithBoolMissing[allshapes.Service, string, allshapes.Item](
				"nonexistent"),
		),
		servicetest.ServiceOnInspect(
			suite.AssertLookupMissing[allshapes.Service, string, allshapes.Item, allshapes.Metadata](
				"nonexistent"),
		),
		servicetest.ServiceOnErr(
			suite.AssertPoisonAccessorNilOnFresh[allshapes.Service](),
		),
	}
}

func benchOpts() []servicetest.ServiceBenchOption {
	return []servicetest.ServiceBenchOption{
		servicetest.ServiceBenchOnGet(
			bench.ReaderHotPath[allshapes.Service, string, allshapes.Item]("seed"),
		),
		servicetest.ServiceBenchOnDescribe(
			bench.PureAllocsWithin[allshapes.Service, string](0),
		),
		servicetest.ServiceBenchOnIsEmpty(
			bench.PredicateAllocsWithin[allshapes.Service](0),
		),
		servicetest.ServiceBenchOnErr(
			bench.PoisonAccessorAllocsWithin[allshapes.Service](0),
		),
		servicetest.ServiceBenchOnTouch(
			bench.MutatorAllocsWithin[allshapes.Service, string]("seed", 0),
		),
	}
}

// --- InMemory ---

func TestServiceContract_InMemory(t *testing.T) {
	t.Parallel()
	servicetest.AssertServiceContract(t, inmemoryFactory, suiteOpts()...)
}

func BenchmarkServiceContract_InMemory(b *testing.B) {
	servicetest.BenchmarkServiceContract(b, inmemoryFactory, benchOpts()...)
}

// --- Stub+DelegateTo ---

func TestServiceContract_StubDelegateTo(t *testing.T) {
	t.Parallel()
	servicetest.AssertServiceContract(t, stubFactory, suiteOpts()...)
}

func BenchmarkServiceContract_StubDelegateTo(b *testing.B) {
	servicetest.BenchmarkServiceContract(b, stubFactory, benchOpts()...)
}

// --- Stub+BenchMode (zero-alloc hot path) ---

func BenchmarkServiceContract_StubBenchMode(b *testing.B) {
	servicetest.BenchmarkServiceContract(b, stubBenchFactory, benchOpts()...)
}
