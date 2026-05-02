// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package basic

import (
	"context"
	"errors"
)

// ErrNotFound is returned when an item is not found.
var ErrNotFound = errors.New("basic: not found")

// ErrConflict is returned on duplicate key.
var ErrConflict = errors.New("basic: conflict")

// internalVar is unexported — should not appear in ErrorVars.
var internalVar = errors.New("basic: internal")

// Status represents an item status.
type Status int

const (
	StatusPending Status = iota
	StatusActive
	StatusClosed
)

// Store manages items.
type Store interface {
	// Get retrieves an item by ID.
	Get(ctx context.Context, id string) (Item, error)
	// Put stores an item.
	Put(ctx context.Context, item Item) error
	// Delete removes an item by ID.
	Delete(ctx context.Context, id string) error
	// Find retrieves multiple items by IDs.
	Find(ctx context.Context, ids ...string) ([]Item, error)
}

// Item is a stored value.
type Item struct {
	ID   string
	Name string
	Tags []string
	age  int // unexported field
}
