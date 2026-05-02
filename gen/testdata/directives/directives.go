// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package directives

import (
	"context"
	"errors"
)

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
//
//testkit:immutable
type Item struct {
	//testkit:optional
	// ID uniquely identifies the item.
	ID string

	// Name has no directives.
	Name string
}

// ErrNotFound is returned when the item does not exist.
//
//testkit:sentinel
var ErrNotFound = errors.New("not found")

// ErrConflict is returned on write conflicts.
var ErrConflict = errors.New("conflict")

// Status represents an item status.
type Status int

const (
	//testkit:default
	// StatusPending is the initial status.
	StatusPending Status = iota

	// StatusActive has no directives.
	StatusActive
)
