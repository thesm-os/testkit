// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package basic

import (
	"context"
	"sync"
)

// InMemoryStore is the [Store] companion. Map-backed storage keyed
// by the Get/Put input parameter; goroutine-safe via RWMutex.
//
// Get returns [ErrNotFound] on a miss. Put writes by item.ID — the
// contract's [AssertReturnsForKey] assertion reads by `key`, so the
// e2e factory pre-seeds the map with the contract's expected
// `(key, value)` pair before returning the impl.
type InMemoryStore struct {
	mu    sync.RWMutex
	items map[string]Item
}

// NewInMemoryStore returns an empty companion.
func NewInMemoryStore() *InMemoryStore {
	return &InMemoryStore{items: make(map[string]Item)}
}

// Seed pre-populates the items map. The e2e contract test seeds
// "test-key" → Item{ID: "test-id"} so [AssertReturnsForKey] resolves
// against a known value.
func (s *InMemoryStore) Seed(key string, item Item) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.items[key] = item
}

// Get implements [Store].
func (s *InMemoryStore) Get(ctx context.Context, key string) (Item, error) {
	if err := ctx.Err(); err != nil {
		return Item{}, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	v, ok := s.items[key]
	if !ok {
		return Item{}, ErrNotFound
	}
	return v, nil
}

// Put implements [Store]. Keys by item.ID. Rejects empty-ID writes
// with [ErrConflict] so the //testkit:atomic contract has an
// observable failure path: AssertAtomicNoTrace forces this branch
// with a zero-valued Item and asserts the map's pre-state is
// unchanged.
func (s *InMemoryStore) Put(ctx context.Context, item Item) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if item.ID == "" {
		return ErrConflict
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.items[item.ID] = item
	return nil
}
