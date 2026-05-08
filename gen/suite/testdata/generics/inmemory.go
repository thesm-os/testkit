// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package generics

import (
	"context"
	"fmt"
)

// InMemoryCache implements [Cache] for testing.
type InMemoryCache[K comparable, V any] struct {
	data  map[K]V
	keyFn func(V) K
}

func NewInMemoryCache[K comparable, V any](keyFn func(V) K) *InMemoryCache[K, V] {
	return &InMemoryCache[K, V]{
		data:  make(map[K]V),
		keyFn: keyFn,
	}
}

func (c *InMemoryCache[K, V]) Get(ctx context.Context, key K) (V, error) {
	if ctx != nil {
		if err := ctx.Err(); err != nil {
			var zero V
			return zero, err
		}
	}
	v, ok := c.data[key]
	if !ok {
		var zero V
		return zero, fmt.Errorf("%w: %v", ErrNotFound, key)
	}
	return v, nil
}

func (c *InMemoryCache[K, V]) Put(ctx context.Context, val V) error {
	if ctx != nil {
		if err := ctx.Err(); err != nil {
			return err
		}
	}
	key := c.keyFn(val)
	c.data[key] = val
	return nil
}

func (c *InMemoryCache[K, V]) Delete(ctx context.Context, key K) error {
	if ctx != nil {
		if err := ctx.Err(); err != nil {
			return err
		}
	}
	if _, ok := c.data[key]; !ok {
		return fmt.Errorf("%w: %v", ErrNotFound, key)
	}
	delete(c.data, key)
	return nil
}

func (c *InMemoryCache[K, V]) Len() int {
	return len(c.data)
}
