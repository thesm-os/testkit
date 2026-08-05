// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// provides the [BootOnlyRegistry] reference for
// register-once interfaces. After a key is registered, repeated
// Register calls error and Lookup returns the original value.

package ref

import (
	"context"
	"sync"
)

// BootOnlyRegistry is a register-once map. Construct with [NewBootOnlyRegistry].
// Thread-safe.
type BootOnlyRegistry[K comparable, V any] struct {
	mu        sync.Mutex
	entries   map[K]V
	duplicate error
}

// NewBootOnlyRegistry constructs an empty [BootOnlyRegistry]. duplicate is the
// error returned by Register when the key is already present.
func NewBootOnlyRegistry[K comparable, V any](duplicate error) *BootOnlyRegistry[K, V] {
	return &BootOnlyRegistry[K, V]{
		entries:   make(map[K]V),
		duplicate: duplicate,
	}
}

// Register inserts an entry. Returns the configured duplicate
// error when k is already registered; the stored value is
// preserved unchanged.
func (b *BootOnlyRegistry[K, V]) Register(_ context.Context, k K, v V) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if _, ok := b.entries[k]; ok {
		return b.duplicate
	}
	b.entries[k] = v
	return nil
}

// Lookup returns the value for k plus true when present, or the
// zero value plus false otherwise.
func (b *BootOnlyRegistry[K, V]) Lookup(_ context.Context, k K) (V, bool) {
	b.mu.Lock()
	defer b.mu.Unlock()
	v, ok := b.entries[k]
	return v, ok
}

// Len returns the number of registered entries.
func (b *BootOnlyRegistry[K, V]) Len() int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return len(b.entries)
}
