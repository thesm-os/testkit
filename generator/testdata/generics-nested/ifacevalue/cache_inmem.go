// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package ifacevalue

import (
	"context"
	"errors"
	"io"
)

// ErrNotFound is the sentinel returned for unknown keys.
var ErrNotFound = errors.New("ifacevalue/cache: not found")

// inmem is the [ReaderCache] companion. The framework's io.Reader
// sample falls back to nil (interface types have no non-zero
// canonical sample); the in-mem returns nil for the contract's
// sample key so the Reader baseline observes (nil, nil).
type inmem struct {
	items map[string]io.Reader
}

// NewInMem returns a fresh inmem with the contract sample key
// mapped to nil, matching the framework's io.Reader sample
// default.
func NewInMem() *inmem {
	return &inmem{
		items: map[string]io.Reader{
			"test-key": nil,
		},
	}
}

// Get implements [ReaderCache].
func (m *inmem) Get(ctx context.Context, key string) (io.Reader, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	v, ok := m.items[key]
	if !ok {
		return nil, ErrNotFound
	}
	return v, nil
}
