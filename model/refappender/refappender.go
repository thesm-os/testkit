// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package refappender provides the [MonotonicLog] reference for
// the Appender contract-tier shape. Append returns the offset of
// the newly appended entry; offsets are monotonically increasing
// and gap-free starting from 0.
package refappender

import (
	"context"
	"sync"
)

// MonotonicLog is an append-only sequence of E. Offsets are
// monotonically increasing, gap-free, and start at zero.
type MonotonicLog[E any] struct {
	mu      sync.Mutex
	entries []E
}

// NewMonotonicLog constructs an empty [MonotonicLog].
func NewMonotonicLog[E any]() *MonotonicLog[E] {
	return &MonotonicLog[E]{}
}

// Append records e and returns its offset (zero-indexed).
func (l *MonotonicLog[E]) Append(_ context.Context, e E) (int64, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	off := int64(len(l.entries))
	l.entries = append(l.entries, e)
	return off, nil
}

// At returns the entry at offset i, or false when out of range.
func (l *MonotonicLog[E]) At(i int64) (E, bool) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if i < 0 || i >= int64(len(l.entries)) {
		var zero E
		return zero, false
	}
	return l.entries[i], true
}

// Len returns the number of entries currently stored.
func (l *MonotonicLog[E]) Len() int {
	l.mu.Lock()
	defer l.mu.Unlock()
	return len(l.entries)
}

// Snapshot returns a copy of every entry in append order.
func (l *MonotonicLog[E]) Snapshot() []E {
	l.mu.Lock()
	defer l.mu.Unlock()
	out := make([]E, len(l.entries))
	copy(out, l.entries)
	return out
}
