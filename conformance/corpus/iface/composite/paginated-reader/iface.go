// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package paginatedreader stacks the reader detector with the pagination
// contract on the same method.
//
// It earns a composite fixture because the contract changes what the detector
// generates rather than adding to it. A bare reader's suite asserts one call
// and one result. Once the method is pagination's reader role the same
// signature has to be driven as a loop — call, take the cursor, call again,
// stop at the zero cursor — and the assertions become "no duplicates across
// pages" and "resuming from a page start yields the full-walk suffix", neither
// of which exists for an unpaginated reader.
//
// A generator that reads the axes independently emits a single-call reader
// subtest and a pagination subtest that never runs, which passes.
package paginatedreader

import (
	"context"
)

// Value is the payload the pages carry.
type Value struct{ Key, Body string }

// PaginatedReader is the fixture interface.
//
//testkit:out paginatedreadertest/ pkg=paginatedreadertest
//testkit:stub
//testkit:suite
//testkit:model
type PaginatedReader interface {
	// Page is a reader by signature and pagination's reader role by directive.
	// The cursor parameter is what the contract keys on.
	//testkit:contract pagination role=reader cursor=Cursor
	Page(ctx context.Context, cursor int) (items []Value, next int, err error)
}
