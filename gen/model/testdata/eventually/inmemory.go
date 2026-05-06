// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package eventually

import (
	"context"
	"sync"
)

// InMemoryBuffer implements [Buffer] as a simple key-value map.
type InMemoryBuffer struct {
	mu   sync.Mutex
	data map[string]string
}

// NewInMemoryBuffer returns a ready-to-use Buffer.
func NewInMemoryBuffer() *InMemoryBuffer {
	return &InMemoryBuffer{data: make(map[string]string)}
}

func (b *InMemoryBuffer) Read(_ context.Context, key string) (string, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	v, ok := b.data[key]
	if !ok {
		return "", ErrNotFound
	}
	return v, nil
}

func (b *InMemoryBuffer) Write(_ context.Context, value string) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.data[value] = value
	return nil
}
