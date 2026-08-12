// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package workflowtest holds the generated harness and double for
// [go.thesmos.sh/testkit/conformance/corpus/iface/contract/workflow], and the
// in-memory subject they are run against.
package workflowtest

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"go.thesmos.sh/testkit/conformance/corpus/iface/contract/workflow"
)

// The states the directive's `transitions=Draft>Live` names, plus the one
// before either.
//
// Absent is spelled because a workflow that has not started is not the same as
// one in its first state: the first Run is a transition too, and a subject
// treating them alike admits a second start.
const (
	Absent = ""
	Draft  = "Draft"
	Live   = "Live"
)

// ErrTerminal reports a transition out of the last declared state.
//
// The directive declares `Draft>Live` and stops there, so Live is terminal.
// Advancing past it is what `AUTO-VALID-TRANSITION` refuses, and an unlabelled
// error would leave a caller unable to tell it from a transport failure.
var ErrTerminal = errors.New("workflowtest: no transition out of the last state")

// InMemory is the implementation the generated conformance harness is run
// against.
//
// One state per key, and the transition table as data rather than as a chain of
// conditionals. The directive is the specification; a subject expressing it as
// control flow has the table written twice, and the copy in the code is the one
// that drifts.
type InMemory struct {
	mu    sync.Mutex
	state map[string]string
	next  map[string]string
}

var _ workflow.Contract = (*InMemory)(nil)

// NewInMemory returns a workflow with nothing started.
func NewInMemory() *InMemory {
	return &InMemory{
		state: map[string]string{},
		next:  map[string]string{Absent: Draft, Draft: Live},
	}
}

// Run advances a key to its next declared state.
func (s *InMemory) Run(ctx context.Context, key string) error {
	if err := contextErr(ctx); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	current := s.state[key]
	next, declared := s.next[current]
	if !declared {
		return fmt.Errorf("%w: %s is terminal", ErrTerminal, current)
	}
	s.state[key] = next
	return nil
}

// State reports the key's observed position: the start state for a key
// nothing has run yet, because the observation collapses "not started" into
// Draft — the first Run then reads as a self-transition, and the declared
// table stays the whole observable relation.
func (s *InMemory) State(ctx context.Context, key string) (string, error) {
	if err := contextErr(ctx); err != nil {
		return "", err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if current := s.state[key]; current != Absent {
		return current, nil
	}
	return Draft, nil
}

// contextErr reports a cancelled or expired context, and tolerates a nil one.
//
// Nil is not a legal context and reaches production anyway, through a caller
// that forgot one. Returning an error is a failed request; dereferencing it is
// an outage, which is why the generated check asks only that this does not
// panic.
func contextErr(ctx context.Context) error {
	if ctx == nil {
		return errors.New("workflowtest: nil context")
	}
	return ctx.Err()
}
