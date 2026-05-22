// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package nestedmap

import (
	"context"
	"errors"
	"maps"
)

// ErrNotFound is the sentinel returned for unknown keys.
var ErrNotFound = errors.New("nestedmap: not found")

// inmem is the [MapCache] companion. Returns a fresh map clone per
// Get so consistent-reads sees equal-but-distinct map instances
// (reflect.DeepEqual on maps with the same entries returns true).
type inmem struct {
	items map[string]map[string]int
}

// NewInMem returns a fresh inmem with the suite's sample (key,
// value) pre-seeded. The framework's map[string]int sample is
// `map[string]int{"test-result0": 42}`; the seed mirrors that
// literal.
func NewInMem() *inmem {
	return &inmem{
		items: map[string]map[string]int{
			"test-key": {"test-result0": 42},
		},
	}
}

// Get implements [MapCache]. Returns a clone so the consistent-
// reads contract observes equal-but-independent maps across calls.
func (m *inmem) Get(ctx context.Context, key string) (map[string]int, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	v, ok := m.items[key]
	if !ok {
		return nil, ErrNotFound
	}
	return maps.Clone(v), nil
}
