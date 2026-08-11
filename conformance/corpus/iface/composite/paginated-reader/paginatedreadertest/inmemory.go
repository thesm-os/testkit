// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package paginatedreadertest holds the generated harness and double for
// [go.thesmos.sh/testkit/conformance/corpus/iface/composite/paginated-reader],
// and the in-memory subject they are run against.
package paginatedreadertest

import (
	"context"
	"errors"
	"slices"
	"sync"

	paginatedreader "go.thesmos.sh/testkit/conformance/corpus/iface/composite/paginated-reader"
)

// Start is the cursor a walk begins at, and End is the one it finishes on.
//
// Zero for both, because the signature has no other way to say either: the
// reader takes an int and returns an int, so "begin" and "no more" have to be
// values of it. That collision is the reason the walk terminates on an empty
// page rather than on the cursor alone.
const (
	Start = 0
	End   = 0
)

// pageSize is how many values a page carries.
const pageSize = 2

// ErrUnknownCursor reports a cursor this reader did not issue.
//
// Cursors are opaque tokens rather than offsets, which is the whole of what
// makes resumption meaningful: an offset into a shifting collection resumes
// somewhere different every time, and a token the reader minted resumes where
// it was minted.
var ErrUnknownCursor = errors.New("paginatedreadertest: cursor was not issued by this reader")

// InMemory is the implementation the generated conformance harness is run
// against.
//
// The issued set is the subject rather than bookkeeping. A reader treating the
// cursor as an offset accepts any integer, so `AUTO-PAGINATOR-RESUMABLE` cannot
// distinguish it from one that resumes correctly — and neither can any check
// that only walks forwards.
type InMemory struct {
	mu     sync.Mutex
	values []paginatedreader.Value
	issued map[int]bool
}

var _ paginatedreader.PaginatedReader = (*InMemory)(nil)

// NewInMemory returns a reader over the given values.
func NewInMemory(values ...paginatedreader.Value) *InMemory {
	return &InMemory{values: values, issued: map[int]bool{Start: true}}
}

// Page returns the values at a cursor and the cursor to resume from.
//
// The next cursor is issued before it is handed out, so a caller that keeps one
// across a restart can still use it and one that invents an integer cannot.
func (s *InMemory) Page(
	ctx context.Context, cursor int,
) (items []paginatedreader.Value, next int, err error) {
	if err := contextErr(ctx); err != nil {
		return nil, End, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.issued[cursor] {
		return nil, End, ErrUnknownCursor
	}
	if cursor >= len(s.values) {
		return nil, End, nil
	}
	to := min(cursor+pageSize, len(s.values))
	page := slices.Clone(s.values[cursor:to])
	if to < len(s.values) {
		s.issued[to] = true
		return page, to, nil
	}
	return page, End, nil
}

// contextErr reports a cancelled or expired context, and tolerates a nil one.
//
// Nil is not a legal context and reaches production anyway, through a caller
// that forgot one. Returning an error is a failed request; dereferencing it is
// an outage, which is why the generated check asks only that this does not
// panic.
func contextErr(ctx context.Context) error {
	if ctx == nil {
		return errors.New("paginatedreadertest: nil context")
	}
	return ctx.Err()
}
