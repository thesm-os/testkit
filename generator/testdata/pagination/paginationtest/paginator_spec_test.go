// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package paginationtest_test

import (
	"testing"

	"go.thesmos.sh/testkit/generator/testdata/pagination"
)

// TestPaginatorContract closes the e2e loop on the
// //testkit:pagination directive. The fixture's in-mem branches
// the cursor space so both the Reader baseline (which expects
// List("test-cursor") to return the sampled Page) and the
// pagination subtest (which iterates from "" until empty) pass
// against the same impl.
func TestPaginatorContract(t *testing.T) {
	t.Parallel()
	AssertPaginatorContract(t, func() pagination.Paginator {
		return pagination.NewInMem()
	})
}
