// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package basic

import (
	"context"
	"sync"
)

// InMemoryStore is a simple in-memory implementation of [Store] for testing
// the generated spec suite.
type InMemoryStore struct {
	mu   sync.Mutex
	data map[string]Item
}

// NewInMemoryStore returns a ready-to-use [InMemoryStore].
func NewInMemoryStore() *InMemoryStore {
	return &InMemoryStore{data: make(map[string]Item)}
}

func ctxErr(ctx context.Context) error {
	if ctx == nil {
		return nil
	}
	return ctx.Err()
}

func (s *InMemoryStore) Get(ctx context.Context, id string) (Item, error) {
	if err := ctxErr(ctx); err != nil {
		return Item{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	v, ok := s.data[id]
	if !ok {
		return Item{}, ErrNotFound
	}
	return v, nil
}

func (s *InMemoryStore) Put(ctx context.Context, item Item) error {
	if err := ctxErr(ctx); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data[item.ID] = item
	return nil
}

func (s *InMemoryStore) Delete(ctx context.Context, id string) error {
	if err := ctxErr(ctx); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.data, id)
	return nil
}

func (s *InMemoryStore) Count(_ context.Context) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.data), nil
}

// NonLinearizableStore is thread-safe (mutex-guarded) but not linearizable:
// Get reads from a stale snapshot that is never updated by Put.
// Used for negative concurrent testing.
type NonLinearizableStore struct {
	mu       sync.Mutex
	data     map[string]Item
	snapshot map[string]Item
}

// NewNonLinearizableStore returns a Store that is safe for concurrent
// use but whose Get returns stale data.
func NewNonLinearizableStore() *NonLinearizableStore {
	return &NonLinearizableStore{
		data:     make(map[string]Item),
		snapshot: make(map[string]Item),
	}
}

func (s *NonLinearizableStore) Get(_ context.Context, id string) (Item, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	v, ok := s.snapshot[id]
	if !ok {
		return Item{}, ErrNotFound
	}
	return v, nil
}

func (s *NonLinearizableStore) Put(_ context.Context, item Item) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data[item.ID] = item
	// BUG: snapshot is never updated — Get always returns stale data
	return nil
}

func (s *NonLinearizableStore) Delete(_ context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.data, id)
	return nil
}

func (s *NonLinearizableStore) Count(_ context.Context) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.data), nil
}
