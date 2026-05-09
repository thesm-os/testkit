// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package nestedmaptest_test

import (
	"testing"

	"go.thesmos.sh/testkit/generator/testdata/generics-nested/nestedmap"
)

// TestMapCacheContract closes the e2e loop on the map-typed value
// cache. Verifies the suite generator emits valid Go for
// map[string]int value types.
func TestMapCacheContract(t *testing.T) {
	t.Parallel()
	AssertMapCacheContract(t, func() nestedmap.MapCache {
		return nestedmap.NewInMem()
	})
}
