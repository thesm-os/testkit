// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package noerror

import "context"

type InMemoryCache struct{ data map[string]string }
func NewInMemoryCache() *InMemoryCache {
	return &InMemoryCache{data: make(map[string]string)}
}

func (c *InMemoryCache) Keys(_ context.Context) []string {
	keys := make([]string, 0, len(c.data))
	for k := range c.data { keys = append(keys, k) }
	return keys
}
func (c *InMemoryCache) Count(_ context.Context) int { return len(c.data) }
func (c *InMemoryCache) Lookup(_ context.Context, key string) *string {
	v, ok := c.data[key]; if !ok { return nil }; return &v
}
func (c *InMemoryCache) Clear(_ context.Context) { c.data = make(map[string]string) }
