// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package basic

import (
	"context"
	"errors"
)

//go:generate testkit stub -o storetest/store_stub.gen.go Store

// ErrNotFound is returned when an item is not found.
var ErrNotFound = errors.New("not found")

// ErrConflict is returned on write conflicts.
var ErrConflict = errors.New("conflict")

// Item is a stored value.
type Item struct {
	ID   string
	Name string
}

// Store is a simple CRUD interface. Exercises the most common stub
// pattern: context + args, single/multi-return with error.
type Store interface {
	// Get retrieves an item by ID.
	Get(ctx context.Context, id string) (Item, error)
	// Put stores an item.
	Put(ctx context.Context, item Item) error
	// Delete removes an item by ID.
	Delete(ctx context.Context, id string) error
}
