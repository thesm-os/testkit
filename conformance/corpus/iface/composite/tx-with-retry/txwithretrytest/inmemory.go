// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package txwithretrytest holds the generated harness and double for
// [go.thesmos.sh/testkit/conformance/corpus/iface/composite/tx-with-retry], and
// the in-memory subject they are run against.
package txwithretrytest

import (
	"context"
	"errors"
	"sync"

	txwithretry "go.thesmos.sh/testkit/conformance/corpus/iface/composite/tx-with-retry"
)

// The states a transaction passes through, and the two it ends in.
const (
	Idle       = "idle"
	Open       = "open"
	Committed  = "committed"
	RolledBack = "rolled back"
)

// InMemory is the implementation the generated conformance harness is run
// against.
//
// The transient failure is a countdown rather than a switch, because the mixin
// is `retrysucceeds`: the point is that a later attempt works, so a subject
// failing forever would make the retry meaningless and one failing never would
// make it untested.
type InMemory struct {
	mu        sync.Mutex
	state     string
	transient int
}

var _ txwithretry.TxWithRetry = (*InMemory)(nil)

// NewInMemory returns a transaction that has not begun, whose first n commits
// fail transiently.
func NewInMemory(transientCommits int) *InMemory {
	return &InMemory{state: Idle, transient: transientCommits}
}

// Begin opens the transaction, and refuses to open a second one.
func (s *InMemory) Begin(ctx context.Context) error {
	if err := contextErr(ctx); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.state == Open {
		return txwithretry.ErrClosed
	}
	s.state = Open
	return nil
}

// Commit settles the transaction, after failing transiently as many times as
// the subject was built to.
//
// A transient failure leaves the state Open, which is the answer this fixture
// exists to pin. If a failed commit settled the transaction the retry would be
// a second terminal operation and the tx contract would refuse it — so the
// suite the two classifications imply would fail against an implementation
// doing exactly what it was told.
func (s *InMemory) Commit(ctx context.Context) error {
	if err := contextErr(ctx); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.state != Open {
		return txwithretry.ErrClosed
	}
	if s.transient > 0 {
		s.transient--
		return txwithretry.ErrTransient
	}
	s.state = Committed
	return nil
}

// Rollback settles the transaction the other way, and refuses one that is not
// open.
func (s *InMemory) Rollback(ctx context.Context) error {
	if err := contextErr(ctx); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.state != Open {
		return txwithretry.ErrClosed
	}
	s.state = RolledBack
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
		return errors.New("txwithretrytest: nil context")
	}
	return ctx.Err()
}
