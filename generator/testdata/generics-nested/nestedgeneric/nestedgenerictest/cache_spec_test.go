// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package nestedgenerictest_test

import (
	"testing"

	"go.thesmos.sh/testkit/generator/testdata/generics-nested/nestedgeneric"
)

// TestPageCacheContract closes the e2e loop on the nested-generic
// value cache. Verifies the suite generator emits valid Go for
// Page[Item] value types.
func TestPageCacheContract(t *testing.T) {
	t.Parallel()
	AssertPageCacheContract(t, func() nestedgeneric.PageCache {
		return nestedgeneric.NewInMem()
	})
}
