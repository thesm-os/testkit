// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package cursortest holds the generated harness and double for
// [go.thesmos.sh/testkit/conformance/corpus/iface/contract/cursor], and the
// in-memory subject they are run against.
package cursortest

import (
	"context"
	"errors"
	"sync"

	"go.thesmos.sh/testkit/conformance/corpus/iface/contract/cursor"
)

// ErrClosed reports a read from a cursor that has been closed.
//
// An error rather than a quiet exhaustion, because the two mean different
// things to the caller: exhausted is "you have everything", closed is "you gave
// up the right to ask". A cursor reporting the first for the second hides a
// bug in the caller's own control flow.
var ErrClosed = errors.New("cursortest: cursor is closed")

// InMemory is the implementation the generated conformance harness is run
// against.
//
// A slice and a position. Exhaustion is the zero value with ok false, which is
// what the comma-ok shape means and what distinguishes "no more" from "there
// was a problem" — the third slot carries the second.
type InMemory struct {
	mu     sync.Mutex
	values []cursor.Value
	at     int
	closed bool
}

var _ cursor.Contract = (*InMemory)(nil)

// NewInMemory returns a cursor over the given values.
func NewInMemory(values ...cursor.Value) *InMemory { return &InMemory{values: values} }

// Next yields the value at the cursor and advances, or reports exhaustion.
//
// The context is consulted first, so a cancelled caller does not advance a
// cursor they will not read from. Advancing anyway loses the value for whoever
// holds the cursor next.
func (s *InMemory) Next(ctx context.Context) (cursor.Value, bool, error) {
	if err := contextErr(ctx); err != nil {
		return cursor.Value{}, false, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return cursor.Value{}, false, ErrClosed
	}
	if s.at >= len(s.values) {
		return cursor.Value{}, false, nil
	}
	v := s.values[s.at]
	s.at++
	return v, true, nil
}

// Close releases the cursor, and says nothing about being called twice.
//
// Idempotent because a caller deferring Close and closing early on an error is
// ordinary Go, and a second close reported as a failure turns a correct
// shutdown into a logged incident.
func (s *InMemory) Close(ctx context.Context) error {
	if err := contextErr(ctx); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.closed = true
	return nil
}

// contextErr reports a cancelled or expired context, and tolerates a nil one.
//
// Nil is not a legal context and reaches production anyway, through a caller
// that forgot one. Returning an error is a failed request; dereferencing it is
// an outage, which is why the generated check asks only that this does not
// panic.
func contextErr(ctx context.Context) error {
	if ctx == nil {
		return errors.New("cursortest: nil context")
	}
	return ctx.Err()
}
