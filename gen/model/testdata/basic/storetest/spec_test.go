// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package storetest_test

import (
	"testing"

	"go.thesmos.sh/testkit/gen/model/testdata/basic"
	"go.thesmos.sh/testkit/gen/model/testdata/basic/storetest"
)

func TestInMemoryStoreModel(t *testing.T) {
	t.Parallel()
	// Tier 0: zero consumer code beyond the factory.
	// Auto-synthesized reference (refmap.MapStore), auto-derived actions,
	// auto-derived ReadAfterWrite + CountEqualsReference laws.
	storetest.AssertStoreModel(t, func() basic.Store {
		return basic.NewInMemoryStore()
	})
}
