// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package basic

import (
	"context"
	"errors"
)

//go:generate testkit model -o storetest/store_model.gen.go Store

// ErrNotFound is returned when an item is not found.
var ErrNotFound = errors.New("not found")

// Item is a stored value.
type Item struct {
	ID   string
	Name string
}

// Store is a pure CRUD interface for model generator testing.
// All methods map to shapes that refmap.MapStore satisfies,
// enabling Tier 0 reference synthesis (zero consumer code).
type Store interface {
	//testkit:errors ErrNotFound
	// Get retrieves an item by ID.
	Get(ctx context.Context, id string) (Item, error)

	// Put stores an item.
	Put(ctx context.Context, item Item) error

	// Delete removes an item by ID.
	Delete(ctx context.Context, id string) error

	// Count returns the number of stored items.
	Count(ctx context.Context) (int, error)
}
