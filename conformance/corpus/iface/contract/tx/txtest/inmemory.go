// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package txtest holds the generated harness and double for
// [go.thesmos.sh/testkit/conformance/corpus/iface/contract/tx], and the
// in-memory subject they are run against.
package txtest

import (
	"context"
	"errors"
	"sync"

	"go.thesmos.sh/testkit/conformance/corpus/iface/contract/tx"
)

// The states a transaction passes through, and the two it ends in.
const (
	Idle       = "idle"
	Open       = "open"
	Committed  = "committed"
	RolledBack = "rolled back"
)

// ErrNotOpen reports a terminal operation on a transaction that is not running.
//
// One error for both cases — never begun, and already settled — because from
// the caller's side they are the same mistake: the handle they hold does not
// name a transaction they may act on.
var ErrNotOpen = errors.New("txtest: no open transaction")

// ErrOpen reports a second Begin on a transaction already running.
var ErrOpen = errors.New("txtest: a transaction is already open")

// InMemory is the implementation the generated conformance harness is run
// against.
//
// A state field and nothing else. What the transaction would guard is left out
// deliberately: the contract's roles are Begin, Commit and Rollback, and the
// claim they carry is that exactly one of the two terminal operations runs.
type InMemory struct {
	mu    sync.Mutex
	state string
}

var _ tx.Contract = (*InMemory)(nil)

// NewInMemory returns a transaction that has not begun.
func NewInMemory() *InMemory { return &InMemory{state: Idle} }

// Begin opens the transaction, and refuses to open a second one.
//
// Refusing rather than nesting. A nested Begin would need a stack and a rule
// for which Commit settles which level, and the fixture declares neither — a
// subject inventing one would be testing this package's idea of nesting.
func (s *InMemory) Begin(ctx context.Context) error {
	if err := contextErr(ctx); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.state == Open {
		return ErrOpen
	}
	s.state = Open
	return nil
}

// Commit settles the transaction, and refuses one that is not open.
func (s *InMemory) Commit(ctx context.Context) error { return s.settle(ctx, Committed) }

// Rollback settles the transaction the other way, and refuses one that is not
// open.
func (s *InMemory) Rollback(ctx context.Context) error { return s.settle(ctx, RolledBack) }

// settle is both terminal operations, because they differ only in the state
// they leave behind.
//
// One statement rather than two, so the rule that a settled transaction cannot
// be settled again cannot be written correctly in one and wrongly in the other.
func (s *InMemory) settle(ctx context.Context, to string) error {
	if err := contextErr(ctx); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.state != Open {
		return ErrNotOpen
	}
	s.state = to
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
		return errors.New("txtest: nil context")
	}
	return ctx.Err()
}
