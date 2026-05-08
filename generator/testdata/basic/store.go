// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package basic

import "context"

// Item is the test value type.
type Item struct {
	ID   string
	Name string
}

// Store is a tiny interface used by loader and shape tests.
//
//testkit:idempotent
type Store interface {
	// Get fetches by key.
	//
	//testkit:errors ErrNotFound
	Get(ctx context.Context, key string) (Item, error)

	// Put writes by key.
	//testkit:directive atomic idempotent
	Put(ctx context.Context, item Item) error
}
