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
	if _, ok := s.data[id]; !ok {
		return ErrNotFound
	}
	delete(s.data, id)
	return nil
}

func (s *InMemoryStore) Count(_ context.Context) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.data)
}

func (s *InMemoryStore) Ping(ctx context.Context) error {
	return ctxErr(ctx)
}

func (s *InMemoryStore) LegacyPut(ctx context.Context, item Item) error {
	return s.Put(ctx, item)
}
