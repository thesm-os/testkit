// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package servicetest_test exercises every shape primitive end-to-end
// against a single interface with one method per shape.
package servicetest_test

import (
	"context"
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/gen/suite/testdata/allshapes"
	"go.thesmos.sh/testkit/gen/suite/testdata/allshapes/servicetest"
	"go.thesmos.sh/testkit/suite"
)

func TestAllShapesContract(t *testing.T) {
	t.Parallel()
	factory := func() allshapes.Service {
		s := allshapes.NewInMemoryService()
		_ = s.Put(context.Background(), allshapes.Item{ID: "seed-1", Name: "seed"})
		return s
	}

	servicetest.AssertServiceContract(t, factory,
		servicetest.ServicePrePopulate(func(ctx context.Context, s allshapes.Service) {
			_ = s.Put(ctx, allshapes.Item{ID: "pre-1", Name: "prepopulated"})
		}),

		// Reader: Get
		servicetest.ServiceOnGet(
			suite.AssertReturnsForKey[allshapes.Service, string, allshapes.Item](
				"seed-1", allshapes.Item{ID: "seed-1", Name: "seed"},
			),
			suite.AssertReturnsSentinel[allshapes.Service, string, allshapes.Item](
				"nonexistent", allshapes.ErrNotFound,
			),
			suite.AssertConsistentReads[allshapes.Service, string, allshapes.Item]("seed-1", 3),
			suite.AssertReaderConcurrentSafe[allshapes.Service, string, allshapes.Item]("seed-1", 4, 50),
			suite.AssertReadsAreNonMutating[allshapes.Service, string, allshapes.Item, int](
				"seed-1",
				func(_ context.Context, s allshapes.Service) int { n, _ := s.Count(context.Background()); return n },
			),
		),

		// Writer: Put
		servicetest.ServiceOnPut(
			suite.AssertWriteSucceeds[allshapes.Service, allshapes.Item](
				allshapes.Item{ID: "new-1", Name: "new"},
			),
			suite.AssertWriteIsObservable[allshapes.Service, allshapes.Item, string](
				allshapes.Item{ID: "obs-1", Name: "observable"},
				func(item allshapes.Item) string { return item.ID },
				func(ctx context.Context, s allshapes.Service, id string) (allshapes.Item, error) {
					return s.Get(ctx, id)
				},
			),
		),

		// Deleter: Delete
		servicetest.ServiceOnDelete(
			suite.AssertDeleteSucceeds[allshapes.Service, string]("seed-1"),
			suite.AssertDeleteReturnsNotFound[allshapes.Service, string]("nonexistent", allshapes.ErrNotFound),
		),

		// Aggregator: Count
		servicetest.ServiceOnCount(
			suite.AssertAggregatorBounded[allshapes.Service, int](0, 1000),
			suite.AssertAggregatorConsistent[allshapes.Service, int](3),
		),

		// Lifecycle: Close
		servicetest.ServiceOnClose(
			suite.AssertLifecycleSucceeds[allshapes.Service](),
			suite.AssertLifecycleIdempotent[allshapes.Service](),
		),

		// Pure: Describe
		servicetest.ServiceOnDescribe(
			suite.AssertDeterministic[allshapes.Service, string](3),
			suite.AssertNoSideEffects[allshapes.Service, string, int](
				func(s allshapes.Service) int { n, _ := s.Count(context.Background()); return n },
			),
		),

		// Predicate: IsEmpty
		servicetest.ServiceOnIsEmpty(
			suite.AssertPredicateConsistent[allshapes.Service](3),
			suite.AssertPredicateReturns[allshapes.Service](false),
		),

		// StreamReader: List
		servicetest.ServiceOnList(
			suite.AssertStreamCompletes[allshapes.Service, allshapes.Item](),
			suite.AssertStreamRespectsBreak[allshapes.Service, allshapes.Item](),
			suite.AssertStreamReentrant[allshapes.Service, allshapes.Item](),
		),

		// Cross-method
		servicetest.ServiceOnAll(
			suite.AssertReadAfterWrite[allshapes.Service, string, allshapes.Item](
				allshapes.Item{ID: "cross-1", Name: "cross"},
				func(ctx context.Context, s allshapes.Service, item allshapes.Item) error { return s.Put(ctx, item) },
				func(ctx context.Context, s allshapes.Service, id string) (allshapes.Item, error) {
					return s.Get(ctx, id)
				},
				func(item allshapes.Item) string { return item.ID },
			),
			suite.AssertDeleteRemovesValue[allshapes.Service, string, allshapes.Item](
				allshapes.Item{ID: "del-1", Name: "delete-me"},
				func(ctx context.Context, s allshapes.Service, item allshapes.Item) error { return s.Put(ctx, item) },
				func(ctx context.Context, s allshapes.Service, id string) error { return s.Delete(ctx, id) },
				func(ctx context.Context, s allshapes.Service, id string) (allshapes.Item, error) {
					return s.Get(ctx, id)
				},
				func(item allshapes.Item) string { return item.ID },
				allshapes.ErrNotFound,
			),
		),

		// Custom
		servicetest.ServiceCustom("describe returns non-empty", func(t *testing.T, s allshapes.Service) {
			desc := s.Describe()
			testkit.True(t, len(desc) > 0, "describe must return non-empty string")
		}),
	)
}
