// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package pagination

import (
	"context"
)

// InMem is the [Paginator] companion. Two contracts run against
// this type:
//
//   - The Reader baseline (Paginator.List is shape-classified as
//     Reader: ctx + K → V + error). Its "returns for key
//     'test-cursor'" subtest calls List("test-cursor") and expects
//     [Page]{Cursor: "test-cursor"} (the framework's struct
//     sample populates the first basic field with "test-<param>").
//
//   - The //testkit:pagination subtest, which iterates the cursor
//     starting at "" until it returns "" again, and asserts no item
//     is yielded twice.
//
// Both work against the same impl as long as the cursor space is
// branched: "test-cursor" returns the sampled Page; "" returns
// page 1 with a "page2" cursor; "page2" returns page 2 with ""
// (end). Three calls drain the corpus deterministically.
// InMem is the [Paginator] companion implementation.
type InMem struct{}

// NewInMem returns a fresh paginator. State is fixed at the
// branches in [List]; nothing to configure.
func NewInMem() *InMem {
	return &InMem{}
}

// List implements [Paginator]. See the InMem doc comment for the
// branches each cursor takes.
func (*InMem) List(ctx context.Context, cursor string) (Page, error) {
	if err := ctx.Err(); err != nil {
		return Page{}, err
	}
	switch cursor {
	case "test-cursor":
		// Reader baseline: List("test-cursor") returns the sample
		// Page{Cursor: "test-cursor"} (the struct sample's first
		// basic field).
		return Page{Cursor: "test-cursor"}, nil
	case "":
		// Pagination start: yield page 1 with a continuation
		// cursor pointing at page 2.
		return Page{
			Items:  []Item{{ID: "item-1"}, {ID: "item-2"}},
			Cursor: "page2",
		}, nil
	case "page2":
		// Pagination end: yield page 2 with an empty cursor
		// signaling no more pages.
		return Page{
			Items:  []Item{{ID: "item-3"}},
			Cursor: "",
		}, nil
	default:
		// Unknown cursor — empty page, terminating cursor.
		return Page{}, nil
	}
}
