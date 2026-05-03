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

//go:generate testkit stub -o storetest/store_stub.gen.go Store

// Store manages items.
type Store interface {
	//testkit:errors ErrNotFound ErrConflict
	// Get retrieves an item by ID.
	Get(ctx context.Context, id string) (Item, error)

	//testkit:errors ErrConflict
	// Put stores an item.
	Put(ctx context.Context, item Item) error

	//testkit:deprecated PutBatch
	// Delete is deprecated in favor of PutBatch.
	Delete(ctx context.Context, id string) error
}

// Item is a stored value.
type Item struct {
	ID   string
	Name string
}
