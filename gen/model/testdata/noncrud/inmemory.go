// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package noncrud

import "context"

// InMemoryCloser implements [Closer] for model testing.
type InMemoryCloser struct{ closed bool }

// NewInMemoryCloser returns a ready-to-use [InMemoryCloser].
func NewInMemoryCloser() *InMemoryCloser { return &InMemoryCloser{} }

func (c *InMemoryCloser) Close(_ context.Context) error {
	c.closed = true
	return nil
}

func (c *InMemoryCloser) Ping(ctx context.Context) error {
	if ctx != nil {
		return ctx.Err()
	}
	return nil
}
