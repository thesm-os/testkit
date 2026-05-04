// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package composed exercises Pillar 2: composed multi-interface
// state-machine testing. Store and Ledger are tested together with
// a cross-interface law verifying that every Store.Put produces a
// corresponding Ledger entry.
package composed

import (
	"context"
	"errors"
)

// ErrNotFound is returned when an item is not found.
var ErrNotFound = errors.New("not found")

// Item is a stored value.
type Item struct {
	ID   string
	Name string
}

// Entry is a ledger entry recording a write.
type Entry struct {
	ItemID string
	Action string // "put" or "delete"
}

// Store is a CRUD interface.
type Store interface {
	Get(ctx context.Context, id string) (Item, error)
	Put(ctx context.Context, item Item) error
	Delete(ctx context.Context, id string) error
}

// Ledger is an append-only log of write operations.
type Ledger interface {
	Append(ctx context.Context, entry Entry) error
	Len(ctx context.Context) (int, error)
}
