// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package nestedgeneric

import (
	"context"
	"errors"
)

// ErrNotFound is the sentinel returned for unknown keys.
var ErrNotFound = errors.New("nestedgeneric/cache: not found")

// inmem is the [PageCache] companion. The framework's Page[Item]
// sample populates Cursor (the first basic-typed field) with
// "test-cursor"; the seed mirrors that literal so the contract
// observes equal values.
type inmem struct {
	items map[string]Page[Item]
}

// NewInMem returns a fresh inmem with the suite's sample (key,
// value) pre-seeded.
func NewInMem() *inmem {
	return &inmem{
		items: map[string]Page[Item]{
			"test-key": {Cursor: "test-cursor"},
		},
	}
}

// Get implements [PageCache].
func (m *inmem) Get(ctx context.Context, key string) (Page[Item], error) {
	if err := ctx.Err(); err != nil {
		return Page[Item]{}, err
	}
	v, ok := m.items[key]
	if !ok {
		return Page[Item]{}, ErrNotFound
	}
	return v, nil
}
