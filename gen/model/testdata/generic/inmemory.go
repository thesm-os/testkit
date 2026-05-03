// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package generic

import (
	"context"
	"sync"
)

// InMemoryRepository implements [ItemRepository] for model testing.
type InMemoryRepository struct {
	mu   sync.Mutex
	data map[string]Item
}

// NewInMemoryRepository returns a ready-to-use [InMemoryRepository].
func NewInMemoryRepository() *InMemoryRepository {
	return &InMemoryRepository{data: make(map[string]Item)}
}

func (r *InMemoryRepository) Get(_ context.Context, k string) (Item, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	v, ok := r.data[k]
	if !ok {
		return Item{}, ErrNotFound
	}
	return v, nil
}

func (r *InMemoryRepository) Put(_ context.Context, v Item) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.data[v.ID] = v
	return nil
}

func (r *InMemoryRepository) Delete(_ context.Context, k string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.data, k)
	return nil
}

func (r *InMemoryRepository) Count(_ context.Context) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.data), nil
}
