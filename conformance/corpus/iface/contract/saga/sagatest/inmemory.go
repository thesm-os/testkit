// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package sagatest holds the generated harness and double for
// [go.thesmos.sh/testkit/conformance/corpus/iface/contract/saga], and the
// in-memory subject they are run against.
package sagatest

import (
	"context"
	"errors"
	"slices"
	"sync"

	"go.thesmos.sh/testkit/conformance/corpus/iface/contract/saga"
)

// ErrNotApplied reports a compensation for a step that never ran.
//
// Reported rather than ignored, because a saga unwinding a step it never
// applied is a coordinator that lost track of where it was — and the
// compensation it runs next will be for a step somebody else owns.
var ErrNotApplied = errors.New("sagatest: no applied step to compensate")

// InMemory is the implementation the generated conformance harness is run
// against.
//
// The applied steps are kept in order, because `AUTO-SAGA-FULL-COMPENSATION` is
// a claim about unwinding: every step that ran has to be undone, and a set
// would lose which one to undo first.
type InMemory struct {
	mu      sync.Mutex
	applied []saga.Value
}

var _ saga.Contract = (*InMemory)(nil)

// NewInMemory returns a saga with nothing applied.
func NewInMemory() *InMemory { return &InMemory{} }

// Step applies a value and records that it was applied.
func (s *InMemory) Step(ctx context.Context, v saga.Value) error {
	if err := contextErr(ctx); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.applied = append(s.applied, v)
	return nil
}

// Compensate undoes a step that was applied.
//
// Keyed on the value rather than positional, because the fixture's signature
// says so: a coordinator naming which step to unwind is compensating a
// particular one, and undoing the most recent regardless would unwind somebody
// else's work when two sagas interleave.
func (s *InMemory) Compensate(ctx context.Context, v saga.Value) error {
	if err := contextErr(ctx); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	at := slices.Index(s.applied, v)
	if at < 0 {
		return ErrNotApplied
	}
	s.applied = slices.Delete(s.applied, at, at+1)
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
		return errors.New("sagatest: nil context")
	}
	return ctx.Err()
}
