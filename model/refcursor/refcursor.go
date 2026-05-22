// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package refcursor provides the [BoundedCursor] reference for the
// Cursor composite-tier shape. Next yields each preloaded element
// exactly once until exhaustion; Close is idempotent;
// Next-after-Close returns the configured sentinel.
package refcursor

import (
	"context"
	"sync"
)

// BoundedCursor is a single-pass iterator over a preloaded slice of
// V. Construct with [NewBoundedCursor]. Thread-safe.
type BoundedCursor[V any] struct {
	mu        sync.Mutex
	items     []V
	index     int
	closed    bool
	closedErr error
}

// NewBoundedCursor constructs a [BoundedCursor] over items.
// closedErr is returned by Next when the cursor has been closed.
func NewBoundedCursor[V any](items []V, closedErr error) *BoundedCursor[V] {
	return &BoundedCursor[V]{
		items:     items,
		closedErr: closedErr,
	}
}

// Next returns the next item plus true, or the zero value plus
// false when exhausted. After Close, Next returns closedErr.
func (c *BoundedCursor[V]) Next(_ context.Context) (V, bool, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		var zero V
		return zero, false, c.closedErr
	}
	if c.index >= len(c.items) {
		var zero V
		return zero, false, nil
	}
	v := c.items[c.index]
	c.index++
	return v, true, nil
}

// Close marks the cursor as closed. Idempotent: a second Close
// returns nil with no effect.
func (c *BoundedCursor[V]) Close(_ context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.closed = true
	return nil
}

// IsClosed reports whether Close has been called.
func (c *BoundedCursor[V]) IsClosed() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.closed
}
