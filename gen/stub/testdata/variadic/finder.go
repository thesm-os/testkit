// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package variadic

import (
	"context"
	"errors"
)

//go:generate testkit stub -o findertest/finder_stub.gen.go Finder

// ErrNotFound is returned when items are not found.
var ErrNotFound = errors.New("not found")

// Finder exercises variadic parameter generation.
type Finder interface {
	// Find retrieves items by IDs. Variadic string parameter.
	Find(ctx context.Context, ids ...string) ([]string, error)
	// Merge combines values. Variadic int parameter.
	Merge(ctx context.Context, values ...int) (int, error)
}
