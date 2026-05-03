// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package servicetest_test

import (
	"context"
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/gen/bench/testdata/allshapes/servicetest"
	"go.thesmos.sh/testkit/gen/suite/testdata/allshapes"
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
			testkit.BenchReaderHotPath[allshapes.Service, string, allshapes.Item]("seed-1"),
			testkit.BenchReaderAllocsWithin[allshapes.Service, string, allshapes.Item]("seed-1", 0),
		),

		// Writer: Put
		servicetest.ServiceBenchOnPut(
			testkit.BenchWriterHotPath[allshapes.Service, allshapes.Item](
				allshapes.Item{ID: "bench-w", Name: "bench"},
			),
		),

		// Deleter: Delete
		servicetest.ServiceBenchOnDelete(
			testkit.BenchDeleterHotPath[allshapes.Service, string]("seed-1"),
		),

		// Aggregator: Count — default hot-path covers measurement; gate allocs.
		servicetest.ServiceBenchOnCount(
			testkit.BenchAggregatorAllocsWithin[allshapes.Service, int](0),
		),

		// Lifecycle: Close — default hot-path covers measurement; gate allocs.
		servicetest.ServiceBenchOnClose(
			testkit.BenchLifecycleAllocsWithin[allshapes.Service](0),
		),

		// Pure: Describe — default hot-path covers measurement; gate allocs.
		servicetest.ServiceBenchOnDescribe(
			testkit.BenchPureAllocsWithin[allshapes.Service, string](0),
		),

		// Predicate: IsEmpty — default hot-path covers measurement; gate allocs.
		servicetest.ServiceBenchOnIsEmpty(
			testkit.BenchPredicateAllocsWithin[allshapes.Service](0),
		),

		// Stream: List
		servicetest.ServiceBenchOnList(
			testkit.BenchStreamHotPath[allshapes.Service, allshapes.Item](),
		),
	)
}
