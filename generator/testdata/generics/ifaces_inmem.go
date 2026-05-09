// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package generics

import (
	"context"
	"sync"
)

// InMemoryHolder is the [Holder] companion. Single-param generic; the
// in-memory map is keyed by string and stores the parameterized V.
type InMemoryHolder[V any] struct {
	mu    sync.RWMutex
	items map[string]V
}

// NewInMemoryHolder returns an empty [Holder] companion.
func NewInMemoryHolder[V any]() *InMemoryHolder[V] {
	return &InMemoryHolder[V]{items: make(map[string]V)}
}

// Get implements [Holder].
func (c *InMemoryHolder[V]) Get(ctx context.Context, key string) (V, error) {
	if err := ctx.Err(); err != nil {
		var zero V
		return zero, err
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	v, ok := c.items[key]
	if !ok {
		var zero V
		return zero, ErrNotFound
	}
	return v, nil
}

// Put implements [Holder].
func (c *InMemoryHolder[V]) Put(ctx context.Context, key string, value V) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.items[key] = value
	return nil
}

// Delete implements [Holder].
func (c *InMemoryHolder[V]) Delete(ctx context.Context, key string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.items, key)
	return nil
}

// InMemoryKeyMap is the [KeyMap] companion. Two type params.
type InMemoryKeyMap[K comparable, V any] struct {
	mu    sync.RWMutex
	items map[K]V
}

// NewInMemoryKeyMap returns an empty [KeyMap] companion.
func NewInMemoryKeyMap[K comparable, V any]() *InMemoryKeyMap[K, V] {
	return &InMemoryKeyMap[K, V]{items: make(map[K]V)}
}

// Get implements [KeyMap].
func (s *InMemoryKeyMap[K, V]) Get(ctx context.Context, key K) (V, error) {
	if err := ctx.Err(); err != nil {
		var zero V
		return zero, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	v, ok := s.items[key]
	if !ok {
		var zero V
		return zero, ErrNotFound
	}
	return v, nil
}

// Set implements [KeyMap].
func (s *InMemoryKeyMap[K, V]) Set(ctx context.Context, key K, value V) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.items[key] = value
	return nil
}

// InMemoryTally is the [Tally] companion. Constrained-T param.
type InMemoryTally[T Numeric] struct {
	mu     sync.Mutex
	totals map[string]T
}

// NewInMemoryTally returns an empty [Tally] companion.
func NewInMemoryTally[T Numeric]() *InMemoryTally[T] {
	return &InMemoryTally[T]{totals: make(map[string]T)}
}

// Add implements [Tally].
func (c *InMemoryTally[T]) Add(ctx context.Context, key string, delta T) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.totals[key] += delta
	return nil
}

// Total implements [Tally]. Returns the sum across all keys.
func (c *InMemoryTally[T]) Total(ctx context.Context) (T, error) {
	if err := ctx.Err(); err != nil {
		var zero T
		return zero, err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	var sum T
	for _, v := range c.totals {
		sum += v
	}
	return sum, nil
}
