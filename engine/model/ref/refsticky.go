// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package ref

import (
	"context"
	"sync"
)

// StickyStore is [MapStore] refined by the sticky claim: the first value a
// key successfully resolves to is the one every later Get answers, whatever
// Put recorded in between. A miss pins nothing — only a resolution sticks —
// and a Delete unpins alongside removing, so a re-added key resolves afresh.
//
// Thread-safe: the outer mutex serializes resolution so two concurrent first
// reads pin one value, and it never holds across the embedded store's own
// lock in a way that can deadlock — the inner store calls nothing back.
type StickyStore[K comparable, V any] struct {
	*MapStore[K, V]

	mu       sync.Mutex
	resolved map[K]V
}

// NewStickyStore creates a [StickyStore] with the given key extractor and
// not-found sentinel error.
func NewStickyStore[K comparable, V any](extractKey func(V) K, notFound error) *StickyStore[K, V] {
	return &StickyStore[K, V]{
		MapStore: NewMapStore(extractKey, notFound),
		resolved: make(map[K]V),
	}
}

// Get resolves the key once and keeps the answer. A miss is not a
// resolution: it reports the sentinel and leaves the key free to resolve to
// whatever a later Put records.
func (s *StickyStore[K, V]) Get(ctx context.Context, k K) (V, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if v, pinned := s.resolved[k]; pinned {
		return v, nil
	}
	v, err := s.MapStore.Get(ctx, k)
	if err != nil {
		return v, err
	}
	s.resolved[k] = v
	return v, nil
}

// Delete removes the value and unpins the key: a deleted key's story ends,
// and the next resolution pins afresh.
func (s *StickyStore[K, V]) Delete(ctx context.Context, k K) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.resolved, k)
	return s.MapStore.Delete(ctx, k)
}
