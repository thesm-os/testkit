// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package pointer

import (
	"context"
	"errors"
)

// ErrNotFound is the sentinel returned for unknown keys.
var ErrNotFound = errors.New("pointer/cache: not found")

// inmem is the [ContainerCache] companion. The framework's pointer-
// to-struct sample is `&Container[int]{Value: 42}`; the in-mem
// returns equivalent fresh allocations on each Get so consistent-
// reads sees deep-equal pointers.
type inmem struct {
	items map[string]*Container[int]
}

// NewInMem returns a fresh inmem with the suite's sample (key,
// value) pre-seeded.
func NewInMem() *inmem {
	return &inmem{
		items: map[string]*Container[int]{
			"test-key": {Value: 42},
		},
	}
}

// Get implements [ContainerCache]. Returns a fresh copy of the
// underlying Container so the consistent-reads contract observes
// deep-equal-but-distinct pointer values across calls.
func (m *inmem) Get(ctx context.Context, key string) (*Container[int], error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	v, ok := m.items[key]
	if !ok {
		return nil, ErrNotFound
	}
	cp := *v
	return &cp, nil
}
