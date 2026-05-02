// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package stub is a test fixture for the stub generator.
package stub

import (
	"context"
	"errors"
)

// ErrNotFound is returned when an item is not found.
var ErrNotFound = errors.New("stub: not found")

// ErrConflict is returned on duplicate key.
var ErrConflict = errors.New("stub: conflict")

// Item is a stored value.
type Item struct {
	ID   string
	Data []byte
}

// Store manages items. Exercises the stub generator with multiple
// method signatures: context+args, single return, multi-return,
// error return, no error return.
type Store interface {
	// Get retrieves an item by ID.
	Get(ctx context.Context, id string) (Item, error)

	// Put stores an item.
	Put(ctx context.Context, item Item) error

	// Delete removes an item by ID.
	Delete(ctx context.Context, id string) error

	// List returns all items. No error return — tests that
	// WithFault is only generated for error-returning methods.
	List(ctx context.Context) []Item
}
