// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package stub

import (
	"context"
	"sync"
)

// storeState holds the in-memory state for NewInMemoryStore.
type storeState struct {
	mu    sync.Mutex
	items map[string]Item
}

// NewInMemoryStore creates a Store backed by in-memory state. This is
// the hand-written companion to the generated StoreStub — it provides
// domain logic via WithStore* constructor options.
func NewInMemoryStore(opts ...StoreStubOption) *StoreStub {
	state := &storeState{items: make(map[string]Item)}
	base := []StoreStubOption{
		WithStoreGet(state.get),
		WithStorePut(state.put),
		WithStoreDelete(state.delete),
		WithStoreList(state.list),
	}
	return NewStoreStub(nil, append(base, opts...)...)
}

func (s *storeState) get(_ context.Context, id string) (Item, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	item, ok := s.items[id]
	if !ok {
		return Item{}, ErrNotFound
	}
	return item, nil
}

func (s *storeState) put(_ context.Context, item Item) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.items[item.ID]; ok {
		return ErrConflict
	}
	s.items[item.ID] = item
	return nil
}

func (s *storeState) delete(_ context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.items[id]; !ok {
		return ErrNotFound
	}
	delete(s.items, id)
	return nil
}

func (s *storeState) list(_ context.Context) []Item {
	s.mu.Lock()
	defer s.mu.Unlock()
	result := make([]Item, 0, len(s.items))
	for _, item := range s.items {
		result = append(result, item)
	}
	return result
}
