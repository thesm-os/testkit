// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package writers

import (
	"context"
	"iter"
	"sync"
)

// InMemoryStore implements [Store] for spec testing.
type InMemoryStore struct {
	mu   sync.Mutex
	data map[string]Item
}

// NewInMemoryStore returns a ready-to-use [InMemoryStore].
func NewInMemoryStore() *InMemoryStore {
	return &InMemoryStore{data: make(map[string]Item)}
}

func (s *InMemoryStore) Get(ctx context.Context, id string) (Item, error) {
	if ctx != nil {
		if err := ctx.Err(); err != nil {
			return Item{}, err
		}
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
	if ctx != nil {
		if err := ctx.Err(); err != nil {
			return err
		}
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data[item.ID] = item
	return nil
}

func (s *InMemoryStore) Delete(ctx context.Context, id string) error {
	if ctx != nil {
		if err := ctx.Err(); err != nil {
			return err
		}
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.data[id]; !ok {
		return ErrNotFound
	}
	delete(s.data, id)
	return nil
}

func (s *InMemoryStore) List(ctx context.Context) iter.Seq2[Item, error] {
	return func(yield func(Item, error) bool) {
		s.mu.Lock()
		defer s.mu.Unlock()
		for _, item := range s.data {
			if ctx != nil && ctx.Err() != nil {
				yield(Item{}, ctx.Err())
				return
			}
			if !yield(item, nil) {
				return
			}
		}
	}
}
