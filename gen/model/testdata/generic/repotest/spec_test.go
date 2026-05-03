// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package repotest_test

import (
	"testing"

	"go.thesmos.sh/testkit/gen/model/testdata/generic"
	"go.thesmos.sh/testkit/gen/model/testdata/generic/repotest"
)

func TestInMemoryRepositoryModel(t *testing.T) {
	t.Parallel()

	t.Run("tier 0 generic interface", func(t *testing.T) {
		t.Parallel()
		// Repository[string, Item] instantiated via type alias.
		// Verifies shape detection works through type parameter
		// instantiation, keyfield heuristic finds Item.ID, and
		// refmap.MapStore satisfies the concrete interface.
		repotest.AssertItemRepositoryModel(t, func() generic.ItemRepository {
			return generic.NewInMemoryRepository()
		})
	})
}
