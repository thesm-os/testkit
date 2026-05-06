// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package leakcheck

import (
	"context"
	"sync"
)

// InMemoryStore implements [Store] correctly with no goroutine leaks.
type InMemoryStore struct {
	mu   sync.Mutex
	data map[string]Item
}

// NewInMemoryStore returns a ready-to-use Store.
func NewInMemoryStore() *InMemoryStore {
	return &InMemoryStore{data: make(map[string]Item)}
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

func (s *InMemoryStore) Put(_ context.Context, item Item) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data[item.ID] = item
	return nil
}

func (s *InMemoryStore) Delete(_ context.Context, id string) error {
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

// LeakyStore implements [Store] but leaks a goroutine on every Put.
type LeakyStore struct {
	mu    sync.Mutex
	data  map[string]Item
	leaks []chan struct{} // leaked goroutines block on these
}

// NewLeakyStore returns a Store that leaks goroutines.
func NewLeakyStore() *LeakyStore {
	return &LeakyStore{data: make(map[string]Item)}
}

func (s *LeakyStore) Get(_ context.Context, id string) (Item, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	v, ok := s.data[id]
	if !ok {
		return Item{}, ErrNotFound
	}
	return v, nil
}

func (s *LeakyStore) Put(_ context.Context, item Item) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data[item.ID] = item

	// BUG: leaks a goroutine on every Put.
	ch := make(chan struct{})
	s.leaks = append(s.leaks, ch)
	go func() { <-ch }()

	return nil
}

func (s *LeakyStore) Delete(_ context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.data, id)
	return nil
}

func (s *LeakyStore) Count(_ context.Context) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.data), nil
}

// Close releases all leaked goroutines. Call in test cleanup.
func (s *LeakyStore) Close() {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, ch := range s.leaks {
		close(ch)
	}
	s.leaks = nil
}
