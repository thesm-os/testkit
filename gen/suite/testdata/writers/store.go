// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package writers exercises the spec generator with Writer, Reader, and
// Stream shapes plus cross-method primitives.
package writers

import (
	"context"
	"errors"
	"iter"
)

//go:generate testkit suite -o storetest/store_spec.gen.go Store

// ErrNotFound is returned when an item is not found.
var ErrNotFound = errors.New("not found")

// Item is a stored value.
type Item struct {
	ID   string
	Name string
}

// Store exercises Writer + Reader + Stream + Deleter shapes.
type Store interface {
	//testkit:errors ErrNotFound
	// Get retrieves an item by ID.
	Get(ctx context.Context, id string) (Item, error)

	// Put stores an item.
	Put(ctx context.Context, item Item) error

	//testkit:deleter
	// Delete removes an item by ID.
	Delete(ctx context.Context, id string) error

	// List returns all items.
	List(ctx context.Context) iter.Seq2[Item, error]
}
