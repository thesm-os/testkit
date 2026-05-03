// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package storetest_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/gen/suite/testdata/basic"
	"go.thesmos.sh/testkit/gen/suite/testdata/basic/storetest"
)

func TestInMemoryStoreContract(t *testing.T) {
	t.Parallel()
	factory := func() basic.Store { return basic.NewInMemoryStore() }

	storetest.AssertStoreContract(t, factory,
		storetest.PrePopulate(func(t testing.TB, s basic.Store) {
			_ = s.Put(t.(*testing.T).Context(), basic.Item{ID: "known-1", Name: "test"})
		}),
		storetest.OnGet(
			testkit.AssertReturnsForKey[basic.Store, string, basic.Item]("known-1", basic.Item{ID: "known-1", Name: "test"}),
			testkit.AssertReturnsSentinel[basic.Store, string, basic.Item]("nonexistent", basic.ErrNotFound),
			testkit.AssertConsistentReads[basic.Store, string, basic.Item]("known-1", 3),
		),
	)
}
