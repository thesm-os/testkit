// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package pagination is the testdata fixture exercising the
// //testkit:pagination directive. The Paginator interface declares
// a cursor-based List method; the in-mem walks a fixed corpus and
// emits an empty cursor when the corpus is drained, satisfying the
// suite's AssertPaginates contract (terminates without duplicates).
package pagination

//go:generate testkit suite -o paginationtest/paginator_spec.gen_test.go Paginator

import "context"

// Item is the value the paginator yields.
type Item struct {
	ID string
}

// Page is the cursor-bearing response. Items is the page's slice;
// Cursor is the marker the caller passes back to fetch the next
// page, or "" when no more pages remain. The suite generator
// extracts Items + the named cursor field via reflection.
type Page struct {
	Items  []Item
	Cursor string
}

// Paginator is a single-method cursor-driven reader.
type Paginator interface {
	// List returns one page of items plus a continuation cursor.
	// An empty returned Cursor signals end-of-stream.
	//
	//testkit:pagination Cursor
	List(ctx context.Context, cursor string) (Page, error)
}
