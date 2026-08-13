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
	"maps"
	"sync"

	"go.thesmos.sh/testkit/conformance/corpus/iface/contract/tx"
)

// InMemory is the implementation the generated conformance harness is run
// against: a keyed store whose writes travel through transactions. Staged
// writes live with their handle until Commit applies them whole; an outside
// Get sees only what committed.
type InMemory struct {
	mu        sync.Mutex
	nextID    int64
	staged    map[int64]map[string]tx.Value
	committed map[string]tx.Value
}

var _ tx.Contract = (*InMemory)(nil)

// NewInMemory returns a store with no transaction open.
func NewInMemory() *InMemory {
	return &InMemory{
		staged:    map[int64]map[string]tx.Value{},
		committed: map[string]tx.Value{},
	}
}

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
	s.staged[s.nextID] = map[string]tx.Value{}
	return tx.Tx{ID: s.nextID}, nil
}

// PutInTx stages a write under the handle's transaction — the subject's own
// staging API, deliberately off the interface: the contract states begin,
// settle and observe, and how a store stages is its own business. The
// generated TxPut door is armed with exactly this.
func (s *InMemory) PutInTx(h tx.Tx, key string, v tx.Value) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	stage, open := s.staged[h.ID]
	if !open {
		return fmt.Errorf("txtest: transaction %d: %w", h.ID, tx.ErrTxClosed)
	}
	stage[key] = v
	return nil
}

// Commit applies the handle's staged writes whole, and refuses a transaction
// already settled.
func (s *InMemory) Commit(ctx context.Context, h tx.Tx) error {
	if err := contextErr(ctx); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	stage, open := s.staged[h.ID]
	if !open {
		return fmt.Errorf("txtest: transaction %d: %w", h.ID, tx.ErrTxClosed)
	}
	maps.Copy(s.committed, stage)
	delete(s.staged, h.ID)
	return nil
}

// Rollback discards the handle's staged writes, and refuses a transaction
// already settled.
func (s *InMemory) Rollback(ctx context.Context, h tx.Tx) error {
	if err := contextErr(ctx); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, open := s.staged[h.ID]; !open {
		return fmt.Errorf("txtest: transaction %d: %w", h.ID, tx.ErrTxClosed)
	}
	delete(s.staged, h.ID)
	return nil
}

// Get observes the committed state — never anything staged, which is the
// mid-transaction claim.
func (s *InMemory) Get(ctx context.Context, key string) (tx.Value, error) {
	if err := contextErr(ctx); err != nil {
		return tx.Value{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	v, present := s.committed[key]
	if !present {
		return tx.Value{}, fmt.Errorf("txtest: key %q: %w", key, tx.ErrNotFound)
	}
	return v, nil
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
