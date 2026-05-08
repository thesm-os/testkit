// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package voidctx

import "context"

// InMemoryCounter implements [Counter] for testing.
type InMemoryCounter struct {
	name  string
	total int64
}

func NewInMemoryCounter(name string) *InMemoryCounter {
	return &InMemoryCounter{name: name}
}

func (c *InMemoryCounter) Add(_ context.Context, value int64) {
	c.total += value
}

func (c *InMemoryCounter) Name() string { return c.name }
