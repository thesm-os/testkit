// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package txtest holds the generated harness and double for
// [go.thesmos.sh/testkit/conformance/corpus/iface/contract/tx], and the
// in-memory subject they are run against.
package txtest

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"go.thesmos.sh/testkit/conformance/corpus/iface/contract/tx"
)

// InMemory is the implementation the generated conformance harness is run
// against.
//
// A set of open transactions and nothing else. What a transaction would guard
// is left out deliberately: the contract's roles are Begin, Commit and
// Rollback, and the claim they carry is that exactly one of the two terminal
// operations settles each handle.
type InMemory struct {
	mu     sync.Mutex
	nextID int64
	open   map[int64]bool
}

var _ tx.Contract = (*InMemory)(nil)

// NewInMemory returns a store with no transaction open.
func NewInMemory() *InMemory { return &InMemory{open: map[int64]bool{}} }

// Begin opens a transaction and answers its handle.
//
// Every Begin opens a fresh one rather than refusing a second: the handle is
// what names a transaction now, so two open transactions are two handles, not
// a conflict over an implicit current one.
func (s *InMemory) Begin(ctx context.Context) (tx.Tx, error) {
	if err := contextErr(ctx); err != nil {
		return tx.Tx{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.nextID++
	s.open[s.nextID] = true
	return tx.Tx{ID: s.nextID}, nil
}

// Commit settles the handle's transaction, and refuses one already settled.
func (s *InMemory) Commit(ctx context.Context, h tx.Tx) error { return s.settle(ctx, h) }

// Rollback settles the handle's transaction the other way, and refuses one
// already settled.
func (s *InMemory) Rollback(ctx context.Context, h tx.Tx) error { return s.settle(ctx, h) }

// settle is both terminal operations, because they differ only in what they
// leave behind — which this subject deliberately has none of.
//
// One statement rather than two, so the rule that a settled transaction cannot
// be settled again cannot be written correctly in one and wrongly in the other.
// A handle that was never begun settles nothing either, and from the caller's
// side that is the same mistake: the handle does not name an open transaction.
func (s *InMemory) settle(ctx context.Context, h tx.Tx) error {
	if err := contextErr(ctx); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.open[h.ID] {
		return fmt.Errorf("txtest: transaction %d: %w", h.ID, tx.ErrTxClosed)
	}
	delete(s.open, h.ID)
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
