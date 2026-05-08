// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package basic

import (
	"context"
	"fmt"
)

// InMemoryStore implements [Store] for testing.
type InMemoryStore struct {
	data map[string]Item
}

// NewInMemoryStore returns a ready-to-use in-memory store.
func NewInMemoryStore() *InMemoryStore {
	return &InMemoryStore{data: make(map[string]Item)}
}

func (s *InMemoryStore) Get(ctx context.Context, id string) (Item, error) {
	if ctx != nil {
		if err := ctx.Err(); err != nil {
			return Item{}, err
		}
	}
	item, ok := s.data[id]
	if !ok {
		return Item{}, fmt.Errorf("%w: %s", ErrNotFound, id)
	}
	return item, nil
}

func (s *InMemoryStore) Put(ctx context.Context, item Item) error {
	if ctx != nil {
		if err := ctx.Err(); err != nil {
			return err
		}
	}
	s.data[item.ID] = item
	return nil
}

func (s *InMemoryStore) Delete(ctx context.Context, id string) error {
	if ctx != nil {
		if err := ctx.Err(); err != nil {
			return err
		}
	}
	if _, ok := s.data[id]; !ok {
		return fmt.Errorf("%w: %s", ErrNotFound, id)
	}
	delete(s.data, id)
	return nil
}

func (s *InMemoryStore) Find(ctx context.Context, ids ...string) ([]Item, error) {
	if ctx != nil {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
	}
	var result []Item
	for _, id := range ids {
		item, ok := s.data[id]
		if !ok {
			return nil, fmt.Errorf("%w: %s", ErrNotFound, id)
		}
		result = append(result, item)
	}
	return result, nil
}

func (s *InMemoryStore) Count(_ context.Context) int {
	return len(s.data)
}

func (s *InMemoryStore) Ping(ctx context.Context) error {
	if ctx != nil {
		return ctx.Err()
	}
	return nil
}

func (s *InMemoryStore) LegacyPut(ctx context.Context, item Item) error {
	return s.Put(ctx, item)
}
