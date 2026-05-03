// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package storetest_test

import (
	"context"
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/gen/suite/testdata/writers"
	"go.thesmos.sh/testkit/gen/suite/testdata/writers/storetest"
)

func TestInMemoryStoreContract(t *testing.T) {
	t.Parallel()
	factory := func() writers.Store { return writers.NewInMemoryStore() }

	storetest.AssertStoreContract(t, factory,
		storetest.PrePopulate(func(ctx context.Context, s writers.Store) {
			_ = s.Put(ctx, writers.Item{ID: "known-1", Name: "test"})
		}),

		// Reader plug-ins on Get.
		storetest.OnGet(
			testkit.AssertReturnsForKey[writers.Store, string, writers.Item](
				"known-1", writers.Item{ID: "known-1", Name: "test"},
			),
			testkit.AssertReturnsSentinel[writers.Store, string, writers.Item](
				"nonexistent", writers.ErrNotFound,
			),
			testkit.AssertConsistentReads[writers.Store, string, writers.Item]("known-1", 3),
		),

		// Writer plug-ins on Put.
		storetest.OnPut(
			testkit.AssertWriteSucceeds[writers.Store, writers.Item](
				writers.Item{ID: "new-1", Name: "new"},
			),
		),

		// Stream plug-ins on List.
		storetest.OnList(
			testkit.AssertStreamCompletes[writers.Store, writers.Item](),
			testkit.AssertStreamRespectsBreak[writers.Store, writers.Item](),
			testkit.AssertStreamReentrant[writers.Store, writers.Item](),
		),

		// Cross-method: read-after-write.
		storetest.OnAll(
			testkit.AssertReadAfterWrite[writers.Store, string, writers.Item](
				writers.Item{ID: "cross-1", Name: "cross"},
				func(ctx context.Context, s writers.Store, item writers.Item) error {
					return s.Put(ctx, item)
				},
				func(ctx context.Context, s writers.Store, id string) (writers.Item, error) {
					return s.Get(ctx, id)
				},
				func(item writers.Item) string { return item.ID },
			),
			testkit.AssertDeleteRemovesValue[writers.Store, string, writers.Item](
				writers.Item{ID: "del-1", Name: "delete-me"},
				func(ctx context.Context, s writers.Store, item writers.Item) error {
					return s.Put(ctx, item)
				},
				func(ctx context.Context, s writers.Store, id string) error {
					return s.Delete(ctx, id)
				},
				func(ctx context.Context, s writers.Store, id string) (writers.Item, error) {
					return s.Get(ctx, id)
				},
				func(item writers.Item) string { return item.ID },
				writers.ErrNotFound,
			),
		),
	)
}
