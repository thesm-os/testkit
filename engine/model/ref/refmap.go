// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// provides generic in-memory reference implementations
// for model-based testing. [MapStore] covers CRUD-shaped interfaces;
// the generator's Tier 0 reference synthesis emits one line:
//
//	func() Store { return NewMapStore(ItemKey, ErrNotFound) }

package ref

import (
	"context"
	"iter"
	"sync"
)

// MapStore is a generic key-value reference implementation for
// Reader + Writer + Deleter + Aggregator + Stream shaped interfaces.
// Thread-safe via mutex.
type MapStore[K comparable, V any] struct {
	mu         sync.Mutex
	data       map[K]V
	extractKey func(V) K
	notFound   error
}

// NewMapStore creates a [MapStore] with the given key extractor and
// not-found sentinel error.
func NewMapStore[K comparable, V any](extractKey func(V) K, notFound error) *MapStore[K, V] {
	return &MapStore[K, V]{
		data:       make(map[K]V),
		extractKey: extractKey,
		notFound:   notFound,
	}
}

// Get returns the value for the given key, or notFound if absent.
func (m *MapStore[K, V]) Get(_ context.Context, k K) (V, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	v, ok := m.data[k]
	if !ok {
		var zero V
		return zero, m.notFound
	}
	return v, nil
}

// Put stores the value, keyed by extractKey(v).
func (m *MapStore[K, V]) Put(_ context.Context, v V) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.data[m.extractKey(v)] = v
	return nil
}

// Delete removes the value for the given key. No-op if absent.
func (m *MapStore[K, V]) Delete(_ context.Context, k K) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.data, k)
	return nil
}

// Count returns the number of stored values.
func (m *MapStore[K, V]) Count(_ context.Context) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.data), nil
}

// List returns all stored values as an iterator.
func (m *MapStore[K, V]) List(_ context.Context) iter.Seq2[V, error] {
	return func(yield func(V, error) bool) {
		m.mu.Lock()
		defer m.mu.Unlock()
		for _, v := range m.data {
			if !yield(v, nil) {
				return
			}
		}
	}
}
