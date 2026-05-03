// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package servicetest_test

import (
	"context"
	"testing"

	"go.thesmos.sh/testkit/bench"
	"go.thesmos.sh/testkit/gen/bench/testdata/allshapes"
	"go.thesmos.sh/testkit/gen/bench/testdata/allshapes/servicetest"
)

func BenchmarkAllShapesContract(b *testing.B) {
	factory := func() allshapes.Service {
		s := allshapes.NewInMemoryService()
		_ = s.Put(context.Background(), allshapes.Item{ID: "seed-1", Name: "seed"})
		return s
	}

	servicetest.BenchmarkServiceContract(b, factory,
		servicetest.ServiceBenchPrePopulate(func(ctx context.Context, s allshapes.Service) {
			_ = s.Put(ctx, allshapes.Item{ID: "pre-1", Name: "prepopulated"})
		}),

		// Reader: Get
		servicetest.ServiceBenchOnGet(
			bench.ReaderHotPath[allshapes.Service, string, allshapes.Item]("seed-1"),
			bench.ReaderAllocsWithin[allshapes.Service, string, allshapes.Item]("seed-1", 0),
		),

		// Writer: Put
		servicetest.ServiceBenchOnPut(
			bench.WriterHotPath[allshapes.Service, allshapes.Item](
				allshapes.Item{ID: "bench-w", Name: "bench"},
			),
		),

		// Deleter: Delete
		servicetest.ServiceBenchOnDelete(
			bench.DeleterHotPath[allshapes.Service, string]("seed-1"),
		),

		// Aggregator: Count — default hot-path covers measurement; gate allocs.
		servicetest.ServiceBenchOnCount(
			bench.AggregatorAllocsWithin[allshapes.Service, int](0),
		),

		// Lifecycle: Close — default hot-path covers measurement; gate allocs.
		servicetest.ServiceBenchOnClose(
			bench.LifecycleAllocsWithin[allshapes.Service](0),
		),

		// Pure: Describe — default hot-path covers measurement; gate allocs.
		servicetest.ServiceBenchOnDescribe(
			bench.PureAllocsWithin[allshapes.Service, string](0),
		),

		// Predicate: IsEmpty — default hot-path covers measurement; gate allocs.
		servicetest.ServiceBenchOnIsEmpty(
			bench.PredicateAllocsWithin[allshapes.Service](0),
		),

		// Stream: List
		servicetest.ServiceBenchOnList(
			bench.StreamHotPath[allshapes.Service, allshapes.Item](),
		),
	)
}
