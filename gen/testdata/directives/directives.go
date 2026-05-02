// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package directives

import "context"

//go:generate testkit stub -o storetest/in_memory_store.gen.go Store
//go:generate testkit recording -o storetest/recording_store.gen.go Store
//go:generate testkit builder -o storetest/builders.gen.go Item

// Store manages items.
type Store interface {
	//testkit:errors ErrNotFound ErrConflict
	//testkit:idempotent
	// Get retrieves an item by ID.
	Get(ctx context.Context, id string) (Item, error)

	//testkit:errors ErrConflict
	//testkit:concurrent
	// Put stores an item.
	Put(ctx context.Context, item Item) error

	// Delete has no directives.
	Delete(ctx context.Context, id string) error
}

// Item is a stored value.
type Item struct {
	ID string
}
