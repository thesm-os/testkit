// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package refregistry provides the [BootOnly] reference for
// register-once interfaces. After a key is registered, repeated
// Register calls error and Lookup returns the original value.
package refregistry

import (
	"context"
	"sync"
)

// BootOnly is a register-once map. Construct with [NewBootOnly].
// Thread-safe.
type BootOnly[K comparable, V any] struct {
	mu        sync.Mutex
	entries   map[K]V
	duplicate error
}

// NewBootOnly constructs an empty [BootOnly]. duplicate is the
// error returned by Register when the key is already present.
func NewBootOnly[K comparable, V any](duplicate error) *BootOnly[K, V] {
	return &BootOnly[K, V]{
		entries:   make(map[K]V),
		duplicate: duplicate,
	}
}

// Register inserts an entry. Returns the configured duplicate
// error when k is already registered; the stored value is
// preserved unchanged.
func (b *BootOnly[K, V]) Register(_ context.Context, k K, v V) error {
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
func (b *BootOnly[K, V]) Lookup(_ context.Context, k K) (V, bool) {
	b.mu.Lock()
	defer b.mu.Unlock()
	v, ok := b.entries[k]
	return v, ok
}

// Len returns the number of registered entries.
func (b *BootOnly[K, V]) Len() int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return len(b.entries)
}
