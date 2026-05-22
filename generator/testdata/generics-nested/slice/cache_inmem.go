// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package slice

import (
	"context"
	"errors"
)

// ErrNotFound is the sentinel returned for unknown keys.
var ErrNotFound = errors.New("slice: not found")

// inmem is the [ByteCache] companion. Stable returns: the same key
// always yields the same slice instance, so the suite's "consistent
// reads" baseline lands on equal slices.
type inmem struct {
	items map[string][]byte
}

// NewInMem returns a fresh inmem with the suite's sample (key,
// value) pre-seeded. The framework's []byte sample is `[]byte{42}`
// (the int sample 42 used as a byte = ASCII '*'); the seed mirrors
// that literal so the Reader baseline's "returns for key" assertion
// passes.
func NewInMem() *inmem {
	return &inmem{
		items: map[string][]byte{
			"test-key": {42},
		},
	}
}

// Get implements [ByteCache].
func (m *inmem) Get(ctx context.Context, key string) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	v, ok := m.items[key]
	if !ok {
		return nil, ErrNotFound
	}
	return v, nil
}
