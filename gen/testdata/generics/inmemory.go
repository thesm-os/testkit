// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package generics

import (
	"context"
	"fmt"
	"iter"
)

// InMemoryCache implements [Cache] for testing.
type InMemoryCache[K comparable, V any] struct {
	data  map[K]V
	keyFn func(V) K
}

// NewInMemoryCache returns a ready-to-use in-memory cache.
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

func (c *InMemoryCache[K, V]) Count(ctx context.Context) (int, error) {
	if ctx != nil {
		if err := ctx.Err(); err != nil {
			return 0, err
		}
	}
	return len(c.data), nil
}

func (c *InMemoryCache[K, V]) Len() int {
	return len(c.data)
}

func (c *InMemoryCache[K, V]) Scan(ctx context.Context) iter.Seq2[V, error] {
	return func(yield func(V, error) bool) {
		if ctx != nil {
			if err := ctx.Err(); err != nil {
				var zero V
				yield(zero, err)
				return
			}
		}
		for _, v := range c.data {
			if !yield(v, nil) {
				return
			}
		}
	}
}

func (c *InMemoryCache[K, V]) Load(_ context.Context, key K) (V, bool) {
	v, ok := c.data[key]
	return v, ok
}
