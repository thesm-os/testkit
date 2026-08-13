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
	"strings"
	"sync"

	"go.thesmos.sh/testkit/conformance/corpus/iface/contract/saga"
)

// ErrNotApplied reports a compensation for a step that never ran.
//
// Reported rather than ignored, because a saga unwinding a step it never
// applied is a coordinator that lost track of where it was — and the
// compensation it runs next will be for a step somebody else owns.
var ErrNotApplied = errors.New("sagatest: no applied step to compensate")

// ErrAlreadyApplied reports a step for a key the saga already stepped.
//
// A refusal rather than a repeat, and it is what makes the compensation law
// reachable: the generated run steps drawn values until one fails, and drawn
// keys collide with the pinned pool by design — so the failing step the law
// needs arrives through the interface instead of through a rigged subject.
var ErrAlreadyApplied = errors.New("sagatest: a step for that key is already applied")

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

// Step applies a value and records that it was applied, refusing a key it
// already stepped.
func (s *InMemory) Step(ctx context.Context, v saga.Value) error {
	if err := contextErr(ctx); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if slices.ContainsFunc(s.applied, func(a saga.Value) bool { return a.Key == v.Key }) {
		return ErrAlreadyApplied
	}
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

// State reports the applied-step fingerprint, in order.
//
// Joined with separators no drawn payload is likely to carry, because the
// fingerprint's whole job is that two different applied sequences never spell
// the same string — a collision here is a compensation defect the law would
// wave through.
func (s *InMemory) State(ctx context.Context) (string, error) {
	if err := contextErr(ctx); err != nil {
		return "", err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	parts := make([]string, len(s.applied))
	for i, v := range s.applied {
		parts[i] = v.Key + "\x00" + v.Body
	}
	return strings.Join(parts, "\x1f"), nil
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
