// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package basic

//go:generate testkit stub -o storetest/store.gen.go Store
//go:generate testkit suite -o storetest/store_spec.gen_test.go Store
//go:generate testkit bench -o storetest/store_bench.gen.go Store

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

	// Put writes by key. Empty ID is rejected so the atomic
	// contract has an observable failure path.
	//
	//testkit:errors ErrConflict
	//testkit:directive atomic idempotent
	Put(ctx context.Context, item Item) error
}
