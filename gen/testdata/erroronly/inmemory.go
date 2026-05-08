// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package erroronly

import (
	"context"
	"sync"
)

// InMemoryCloser implements [Closer] for spec testing.
type InMemoryCloser struct {
	mu     sync.Mutex
	opened bool
}

// NewInMemoryCloser returns a ready-to-use [InMemoryCloser].
func NewInMemoryCloser() *InMemoryCloser { return &InMemoryCloser{} }

func (c *InMemoryCloser) Open(ctx context.Context) error {
	if ctx != nil {
		if err := ctx.Err(); err != nil {
			return err
		}
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.opened = true
	return nil
}

func (c *InMemoryCloser) Close(ctx context.Context) error {
	if ctx != nil {
		if err := ctx.Err(); err != nil {
			return err
		}
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.opened {
		return ErrClosed
	}
	c.opened = false
	return nil
}
