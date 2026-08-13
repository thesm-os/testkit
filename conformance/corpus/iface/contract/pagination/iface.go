// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package pagination is the contract-axis fixture for the pagination contract:
// a reader whose protocol cursors through pages.
//
// Every role the contract declares is present, because a contract is a
// multi-callable protocol and a fixture missing a partner exercises the
// validator's failure path rather than the contract itself.
package pagination

import (
	"context"
)

// Value is the payload the contract's roles carry.
type Value struct{ Key, Body string }

// Cursor is where a walk resumes; its zero value is the first page.
type Cursor string

// Contract is the fixture interface.
//
//testkit:out paginationtest/ pkg=paginationtest
//testkit:stub
//testkit:suite
//testkit:model
type Contract interface {
	// Page is the pagination contract's reader role, and hosts the
	// directive that names its partners. It answers one page, where the
	// walk resumes, and whether anything remains — the shape both walk
	// claims need, because a keyed read has no cursor to resume from.
	//testkit:contract pagination role=reader cursor=Cursor
	Page(ctx context.Context, cur Cursor) (items []Value, next Cursor, more bool, err error)

	// Put stores an entry the walk will visit. Without a writer the
	// sequences seed nothing and every walk holds over empty pages.
	Put(ctx context.Context, v Value) error
}
