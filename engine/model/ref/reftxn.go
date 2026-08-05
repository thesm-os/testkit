// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// provides the [SnapshotIsolation] reference for
// the TransactionFunc contract-tier shape. Each transaction sees
// the snapshot taken at Begin; writes are buffered and applied
// atomically on Commit; Rollback discards the buffer.
//
// The reference implements snapshot isolation (G0/G1/G2 anomalies
// are forbidden) by virtue of the simple buffering: writes from
// other transactions land in the committed store, but the active
// transaction always reads from its own snapshot.

package ref

import (
	"context"
	"errors"
	"maps"
	"sync"
)

// ErrTxClosed is returned by operations on a transaction that has
// already been committed or rolled back.
var ErrTxClosed = errors.New("ref: transaction is closed")

// SnapshotIsolation is a key-value store with snapshot-isolated
// transactions. Construct with [NewSnapshotIsolation]. Thread-safe
// for both store-level and transaction-level operations.
type SnapshotIsolation[K comparable, V any] struct {
	mu        sync.Mutex
	committed map[K]V
	notFound  error
}

// NewSnapshotIsolation constructs an empty store. notFound is
// returned by [SnapshotTx.Get] for absent keys.
func NewSnapshotIsolation[K comparable, V any](notFound error) *SnapshotIsolation[K, V] {
	return &SnapshotIsolation[K, V]{
		committed: make(map[K]V),
		notFound:  notFound,
	}
}

// Begin returns a new transaction. The transaction sees the
// committed store as of this call; subsequent commits from other
// transactions are invisible until the next Begin.
func (s *SnapshotIsolation[K, V]) Begin(_ context.Context) (*SnapshotTx[K, V], error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	snap := make(map[K]V, len(s.committed))
	maps.Copy(snap, s.committed)
	return &SnapshotTx[K, V]{
		store:    s,
		snapshot: snap,
		buffer:   make(map[K]V),
	}, nil
}

// Get reads the committed value for k. notFound for absent.
func (s *SnapshotIsolation[K, V]) Get(_ context.Context, k K) (V, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	v, ok := s.committed[k]
	if !ok {
		var zero V
		return zero, s.notFound
	}
	return v, nil
}

// SnapshotTx is a snapshot-isolated transaction. Reads come from the
// snapshot taken at Begin plus the transaction's own write buffer.
type SnapshotTx[K comparable, V any] struct {
	store    *SnapshotIsolation[K, V]
	snapshot map[K]V
	buffer   map[K]V
	closed   bool
}

// Get returns the value for k, observing the transaction's own
// uncommitted writes first then the snapshot.
func (t *SnapshotTx[K, V]) Get(_ context.Context, k K) (V, error) {
	if t.closed {
		var zero V
		return zero, ErrTxClosed
	}
	if v, ok := t.buffer[k]; ok {
		return v, nil
	}
	if v, ok := t.snapshot[k]; ok {
		return v, nil
	}
	var zero V
	return zero, t.store.notFound
}

// Put buffers the write; visible inside the transaction but not
// to other transactions until Commit.
func (t *SnapshotTx[K, V]) Put(_ context.Context, k K, v V) error {
	if t.closed {
		return ErrTxClosed
	}
	t.buffer[k] = v
	return nil
}

// Commit applies every buffered write to the committed store
// atomically. Idempotent on a previously-closed transaction.
func (t *SnapshotTx[K, V]) Commit(_ context.Context) error {
	if t.closed {
		return ErrTxClosed
	}
	t.store.mu.Lock()
	defer t.store.mu.Unlock()
	maps.Copy(t.store.committed, t.buffer)
	t.closed = true
	return nil
}

// Rollback discards the buffered writes.
func (t *SnapshotTx[K, V]) Rollback(_ context.Context) error {
	if t.closed {
		return ErrTxClosed
	}
	t.closed = true
	return nil
}
