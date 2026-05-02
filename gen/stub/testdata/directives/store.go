// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package directives

import (
	"context"
	"errors"
)

// ErrNotFound is returned when the item does not exist.
var ErrNotFound = errors.New("not found")

// ErrConflict is returned on write conflicts.
var ErrConflict = errors.New("conflict")

// Store manages items.
type Store interface {
	//testkit:errors ErrNotFound ErrConflict
	// Get retrieves an item by ID.
	Get(ctx context.Context, id string) (Item, error)

	//testkit:errors ErrConflict
	// Put stores an item.
	Put(ctx context.Context, item Item) error

	// Delete has no directives.
	Delete(ctx context.Context, id string) error
}

// Item is a stored value.
type Item struct {
	ID   string
	Name string
}
