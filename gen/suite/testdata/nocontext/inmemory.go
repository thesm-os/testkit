// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package nocontext

import "sync"

// InMemoryCache is a simple in-memory implementation of [Cache].
type InMemoryCache struct {
	mu   sync.Mutex
	data map[string]string
}

// NewInMemoryCache returns a ready-to-use [InMemoryCache].
func NewInMemoryCache() *InMemoryCache {
	return &InMemoryCache{data: make(map[string]string)}
}

func (c *InMemoryCache) Get(key string) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	v, ok := c.data[key]
	if !ok {
		return "", ErrMiss
	}
	return v, nil
}

func (c *InMemoryCache) Set(key string, value string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.data[key] = value
	return nil
}

func (c *InMemoryCache) Len() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.data)
}
