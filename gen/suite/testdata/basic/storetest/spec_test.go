// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package storetest_test

import (
	"context"
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/gen/suite/testdata/basic"
	"go.thesmos.sh/testkit/gen/suite/testdata/basic/storetest"
	"go.thesmos.sh/testkit/suite"
)

func TestInMemoryStoreContract(t *testing.T) {
	t.Parallel()
	factory := func() basic.Store { return basic.NewInMemoryStore() }

	storetest.AssertStoreContract(t, factory,
		storetest.StorePrePopulate(func(ctx context.Context, s basic.Store) {
			_ = s.Put(ctx, basic.Item{ID: "known-1", Name: "test"})
		}),
		storetest.StoreOnGet(
			suite.AssertReturnsForKey[basic.Store, string, basic.Item]("known-1", basic.Item{ID: "known-1", Name: "test"}),
			suite.AssertReturnsSentinel[basic.Store, string, basic.Item]("nonexistent", basic.ErrNotFound),
			suite.AssertConsistentReads[basic.Store, string, basic.Item]("known-1", 3),
		),
		storetest.StoreOnPut(
			suite.AssertWriteSucceeds[basic.Store, basic.Item](basic.Item{ID: "new-1", Name: "new"}),
		),
		storetest.StoreOnDelete(
			suite.AssertWriteSucceeds[basic.Store, string]("known-1"),
		),
		storetest.StoreOnPing(
			suite.AssertLifecycleSucceeds[basic.Store](),
		),
		storetest.StoreOnCount(func(t *testing.T, s basic.Store) {
			c1 := s.Count(t.Context())
			c2 := s.Count(t.Context())
			testkit.Equal(t, c1, c2, "Count must be deterministic")
		}),
		storetest.StoreOnLegacyPut(
			suite.AssertWriteSucceeds[basic.Store, basic.Item](basic.Item{ID: "legacy", Name: "legacy"}),
		),
		storetest.StoreOnAll(
			suite.AssertReadAfterWrite[basic.Store, string, basic.Item](
				basic.Item{ID: "cross-1", Name: "cross"},
				func(ctx context.Context, s basic.Store, item basic.Item) error { return s.Put(ctx, item) },
				func(ctx context.Context, s basic.Store, id string) (basic.Item, error) { return s.Get(ctx, id) },
				func(item basic.Item) string { return item.ID },
			),
		),
		storetest.StoreCustom("custom subtest", func(t *testing.T, s basic.Store) {
			testkit.NoError(t, s.Put(t.Context(), basic.Item{ID: "c", Name: "custom"}), "custom put")
		}),
	)
}
