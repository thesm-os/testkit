// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package refpaginator provides the [CursorTable] reference for
// the Paginator contract-tier shape. The table is a map plus an
// ordered key list; Page hands out items by stable cursor and
// returns the next cursor (or zero, signaling end).
package refpaginator

import (
	"context"
	"sort"
	"sync"
)

// CursorTable is a key-ordered store paginated by integer cursor.
// Construct with [NewCursorTable]. Thread-safe.
type CursorTable[K comparable, V any] struct {
	mu    sync.Mutex
	data  map[K]V
	keys  []K
	less  func(a, b K) bool
	dirty bool
}

// NewCursorTable constructs a [CursorTable] ordered by less.
func NewCursorTable[K comparable, V any](less func(a, b K) bool) *CursorTable[K, V] {
	return &CursorTable[K, V]{
		data: make(map[K]V),
		less: less,
	}
}

// Put inserts or replaces an entry. Marks the key list dirty so
// the next Page call resorts.
func (t *CursorTable[K, V]) Put(_ context.Context, k K, v V) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	if _, present := t.data[k]; !present {
		t.keys = append(t.keys, k)
		t.dirty = true
	}
	t.data[k] = v
	return nil
}

// Page returns up to pageSize entries starting at cursor (which is
// a stable position index). The returned nextCursor is the index
// of the next unread entry; when zero, the drain has reached the
// end. cursor is interpreted as an int via the consumer's int
// cursor type.
func (t *CursorTable[K, V]) Page(_ context.Context, cursor, pageSize int) ([]V, int, error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.dirty {
		sort.Slice(t.keys, func(i, j int) bool { return t.less(t.keys[i], t.keys[j]) })
		t.dirty = false
	}
	if cursor < 0 {
		cursor = 0
	}
	if cursor >= len(t.keys) {
		return nil, 0, nil
	}
	end := min(cursor+pageSize, len(t.keys))
	out := make([]V, 0, end-cursor)
	for _, k := range t.keys[cursor:end] {
		out = append(out, t.data[k])
	}
	if end >= len(t.keys) {
		return out, 0, nil
	}
	return out, end, nil
}

// Len returns the number of entries in the table.
func (t *CursorTable[K, V]) Len() int {
	t.mu.Lock()
	defer t.mu.Unlock()
	return len(t.data)
}
