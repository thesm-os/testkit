// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package basic

import (
	"context"
	"sync"
)

// InMemorySampler is the [Sampler] companion used by the bench
// wrapper. Pre-seeded with [SampleKey]() → [SampleItem]() so the
// `//testkit:sample` directive's expected (key, value) pair lands
// on a populated entry rather than the not-found path.
type InMemorySampler struct {
	mu    sync.RWMutex
	items map[string]Item
}

// NewInMemorySampler returns a companion seeded with the //testkit:sample
// pair so Lookup hits the success path on the first call.
func NewInMemorySampler() *InMemorySampler {
	s := &InMemorySampler{items: map[string]Item{}}
	s.items[SampleKey()] = SampleItem()
	return s
}

// Lookup implements [Sampler].
func (s *InMemorySampler) Lookup(ctx context.Context, key string) (Item, error) {
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

// Apply implements [Sampler]. Stores by the (key, item) pair.
func (s *InMemorySampler) Apply(ctx context.Context, key string, item Item) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.items[key] = item
	return nil
}
