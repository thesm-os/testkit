// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package stub

import (
	"context"
	"io"
	"sync"
	"time"
)

// storeState holds the in-memory state for NewInMemoryStore.
type storeState struct {
	mu    sync.Mutex
	items map[ID]Item
}

// NewInMemoryStore creates a Store backed by in-memory state. This is
// the hand-written companion to the generated StoreStub — it provides
// domain logic via WithStore* constructor options.
func NewInMemoryStore(opts ...StoreStubOption) *StoreStub {
	state := &storeState{items: make(map[ID]Item)}
	base := []StoreStubOption{
		WithStoreGet(state.get),
		WithStorePut(state.put),
		WithStoreDelete(state.delete),
		WithStoreList(state.list),
		WithStoreCount(state.count),
		WithStoreFind(state.find),
		WithStorePutMany(state.putMany),
		WithStoreImport(state.doImport),
		WithStoreExport(state.doExport),
		WithStoreGetOptional(state.getOptional),
		WithStoreTouch(state.touch),
		WithStorePing(state.ping),
		WithStoreTags(state.tags),
		WithStoreMetadataFor(state.metadataFor),
		WithStoreClose(state.doClose),
	}
	return NewStoreStub(nil, append(base, opts...)...)
}

func (s *storeState) get(_ context.Context, id ID) (Item, error) {
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

func (s *storeState) delete(_ context.Context, id ID) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.items[id]; !ok {
		return ErrNotFound
	}
	delete(s.items, id)
	return nil
}

func (s *storeState) list(_ context.Context, _ ListOptions) (ListResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	items := make([]Item, 0, len(s.items))
	for _, item := range s.items {
		items = append(items, item)
	}
	return ListResult{Items: items, Total: len(items)}, nil
}

func (s *storeState) count(_ context.Context) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.items)
}

func (s *storeState) find(_ context.Context, ids ...ID) ([]Item, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	result := make([]Item, 0, len(ids))
	for _, id := range ids {
		item, ok := s.items[id]
		if !ok {
			return nil, ErrNotFound
		}
		result = append(result, item)
	}
	return result, nil
}

func (s *storeState) putMany(_ context.Context, items ...Item) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, item := range items {
		if _, ok := s.items[item.ID]; ok {
			return ErrConflict
		}
	}
	for _, item := range items {
		s.items[item.ID] = item
	}
	return nil
}

func (s *storeState) doImport(_ context.Context, _ io.Reader) (int, error) {
	return 0, nil
}

func (s *storeState) doExport(_ context.Context, _ io.Writer) error {
	return nil
}

func (s *storeState) getOptional(_ context.Context, id ID) *Item {
	s.mu.Lock()
	defer s.mu.Unlock()
	item, ok := s.items[id]
	if !ok {
		return nil
	}
	return &item
}

func (s *storeState) touch(_ context.Context, id ID) (time.Time, time.Time, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	item, ok := s.items[id]
	if !ok {
		return time.Time{}, time.Time{}, ErrNotFound
	}
	before := item.CreatedAt
	item.CreatedAt = time.Now()
	s.items[id] = item
	return before, item.CreatedAt, nil
}

func (s *storeState) ping(_ context.Context) error {
	return nil
}

func (s *storeState) tags(_ context.Context) []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	seen := make(map[string]bool)
	for _, item := range s.items {
		for _, tag := range item.Tags {
			seen[tag] = true
		}
	}
	result := make([]string, 0, len(seen))
	for tag := range seen {
		result = append(result, tag)
	}
	return result
}

func (s *storeState) metadataFor(_ context.Context, id ID) (map[string]string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	item, ok := s.items[id]
	if !ok {
		return nil, ErrNotFound
	}
	return item.Metadata, nil
}

func (s *storeState) doClose() error {
	return nil
}
