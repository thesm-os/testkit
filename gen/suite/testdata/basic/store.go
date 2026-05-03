// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package basic

import (
	"context"
	"errors"
)

//go:generate testkit suite -o storetest/store_spec.gen.go Store

// ErrNotFound is returned when an item is not found.
var ErrNotFound = errors.New("not found")

// ErrConflict is returned on write conflicts.
var ErrConflict = errors.New("conflict")

// Item is a stored value.
type Item struct {
	ID   string
	Name string
}

// Store exercises the spec generator's directives.
type Store interface {
	//testkit:errors ErrNotFound
	//testkit:ctx
	//testkit:pure
	// Get retrieves an item by ID.
	Get(ctx context.Context, id string) (Item, error)

	//testkit:nilsafe
	//testkit:ctx
	// Put stores an item.
	Put(ctx context.Context, item Item) error

	//testkit:ctx
	// Delete removes an item by ID.
	Delete(ctx context.Context, id string) error

	//testkit:bounded 0 1000
	// Count returns the number of stored items.
	Count(ctx context.Context) int

	//testkit:timeout 5s
	// Ping checks connectivity.
	Ping(ctx context.Context) error

	//testkit:deprecated PutBatch
	// LegacyPut is deprecated in favor of PutBatch.
	LegacyPut(ctx context.Context, item Item) error
}
