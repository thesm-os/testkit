// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package composed

import (
	"context"
	"sync"
)

// InMemoryStore implements [Store].
type InMemoryStore struct {
	mu     sync.Mutex
	data   map[string]Item
	ledger Ledger // wired to ledger for cross-interface coordination
}

// NewInMemoryStore returns a Store that appends to the given Ledger on every Put/Delete.
func NewInMemoryStore(ledger Ledger) *InMemoryStore {
	return &InMemoryStore{data: make(map[string]Item), ledger: ledger}
}

func (s *InMemoryStore) Get(_ context.Context, id string) (Item, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	v, ok := s.data[id]
	if !ok {
		return Item{}, ErrNotFound
	}
	return v, nil
}

func (s *InMemoryStore) Put(ctx context.Context, item Item) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data[item.ID] = item
	return s.ledger.Append(ctx, Entry{ItemID: item.ID, Action: "put"})
}

func (s *InMemoryStore) Delete(ctx context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.data, id)
	return s.ledger.Append(ctx, Entry{ItemID: id, Action: "delete"})
}

// InMemoryLedger implements [Ledger].
type InMemoryLedger struct {
	mu      sync.Mutex
	entries []Entry
}

// NewInMemoryLedger returns a ready-to-use Ledger.
func NewInMemoryLedger() *InMemoryLedger {
	return &InMemoryLedger{}
}

func (l *InMemoryLedger) Append(_ context.Context, entry Entry) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.entries = append(l.entries, entry)
	return nil
}

func (l *InMemoryLedger) Len(_ context.Context) (int, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	return len(l.entries), nil
}
