// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package basic

import (
	"context"
	"sync"
)

// InMemoryPerf is the [Perf] companion used by the bench wrapper.
// Pre-seeded with the synthesized hot-path sample so the
// `//testkit:allocs 0` budget gate observes the success path
// instead of the error-allocation path that would otherwise
// dominate the alloc count.
type InMemoryPerf struct {
	mu    sync.RWMutex
	items map[string]Item
}

// NewInMemoryPerf returns a companion seeded with the bench
// generator's default Reader sample ("test-key" → Item{ID: "test-id"}).
func NewInMemoryPerf() *InMemoryPerf {
	return &InMemoryPerf{
		items: map[string]Item{
			"test-key": {ID: "test-id"},
		},
	}
}

// Hot implements [Perf]. Returns a seeded Item for known keys, or
// [ErrNotFound] for misses. The bench //testkit:allocs 0 directive
// expects the success path to be alloc-free; the test wrapper
// seeds "test-key" so the gate measures the right thing.
func (p *InMemoryPerf) Hot(ctx context.Context, key string) (Item, error) {
	if err := ctx.Err(); err != nil {
		return Item{}, err
	}
	p.mu.RLock()
	defer p.mu.RUnlock()
	v, ok := p.items[key]
	if !ok {
		return Item{}, ErrNotFound
	}
	return v, nil
}
