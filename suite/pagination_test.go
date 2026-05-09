// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package suite_test

import (
	"context"
	"testing"

	"go.thesmos.sh/testkit/suite"
)

// page is a fixture struct matching the pagination contract: an
// `Items` slice plus a configurable cursor field.
type page struct {
	Items    []int
	NextPage string
}

// pager is a fixture impl that yields three pages of [Items], then
// signals termination by returning an empty cursor.
type pager struct{}

func (pager) Next(_ context.Context, cursor string) (page, error) {
	switch cursor {
	case "":
		return page{Items: []int{1, 2, 3}, NextPage: "p2"}, nil
	case "p2":
		return page{Items: []int{4, 5}, NextPage: "p3"}, nil
	case "p3":
		return page{Items: []int{6}, NextPage: ""}, nil
	}
	return page{}, nil
}

func TestAssertPaginates(t *testing.T) {
	t.Parallel()

	t.Run("happy path: cursor terminates, no duplicates", func(t *testing.T) {
		t.Parallel()
		factory := func() pager { return pager{} }
		suite.AssertPaginates[pager, string, page](
			suite.PaginationConfig{CursorField: "NextPage", EmptyCursor: ""},
			func(ctx context.Context, impl pager, cursor string) (page, error) {
				return impl.Next(ctx, cursor)
			},
		)(t, factory)
	})

	t.Run("default Iterations cap is applied (3-page run still works)", func(t *testing.T) {
		t.Parallel()
		// The default cap (1000) is well above the fixture's 3 pages.
		// Configured value of 0 picks the default; we verify by
		// running a successful 3-page iteration.
		factory := func() pager { return pager{} }
		suite.AssertPaginates[pager, string, page](
			suite.PaginationConfig{CursorField: "NextPage", EmptyCursor: "", Iterations: 0},
			func(ctx context.Context, impl pager, cursor string) (page, error) {
				return impl.Next(ctx, cursor)
			},
		)(t, factory)
	})

	t.Run("Items slice with string element types is iterated correctly", func(t *testing.T) {
		t.Parallel()
		factory := func() *strPager { return &strPager{} }
		suite.AssertPaginates[*strPager, string, strPage](
			suite.PaginationConfig{CursorField: "NextPage", EmptyCursor: ""},
			func(ctx context.Context, impl *strPager, cursor string) (strPage, error) {
				return impl.Next(ctx, cursor)
			},
		)(t, factory)
	})
}

// strPager is a string-typed pager used to verify the reflection-
// driven Items extraction handles element types beyond int.
type strPager struct{}

// Next implements the page-returning shape consumed by
// [suite.AssertPaginates] for the strPage element type.
func (*strPager) Next(_ context.Context, cursor string) (strPage, error) {
	if cursor == "" {
		return strPage{Items: []string{"a", "b"}, NextPage: ""}, nil
	}
	return strPage{}, nil
}

// strPage mirrors [page] with string-typed Items so the test
// exercises non-int element handling in the helper.
type strPage struct {
	Items    []string
	NextPage string
}
