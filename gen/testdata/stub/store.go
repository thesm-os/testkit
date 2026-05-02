// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package stub is a test fixture for the stub generator.
package stub

import (
	"context"
	"errors"
	"io"
	"time"
)

// ErrNotFound is returned when an item is not found.
var ErrNotFound = errors.New("stub: not found")

// ErrConflict is returned on duplicate key.
var ErrConflict = errors.New("stub: conflict")

// ID is a named string type for identifiers.
type ID string

// Status represents an item status.
type Status int

const (
	StatusPending Status = iota // Pending
	StatusActive                // Active
	StatusClosed                // Closed
)

// Item is a stored value.
type Item struct {
	ID        ID
	Name      string
	Data      []byte
	Status    Status
	CreatedAt time.Time
	Tags      []string
	Metadata  map[string]string
}

// ListOptions controls pagination and filtering.
type ListOptions struct {
	Limit  int
	Offset int
	Status *Status
}

// ListResult is a paginated result set.
type ListResult struct {
	Items      []Item
	Total      int
	HasMore    bool
	NextCursor string
}

// Store manages items. Exercises the stub generator with real-world
// method signatures covering all common patterns:
//   - Single and multi-return with error
//   - No error return
//   - Named return values
//   - Variadic parameters (string and struct)
//   - Pointer parameters and returns
//   - Named types (ID, Status)
//   - Struct parameters and returns
//   - Slice and map returns
//   - Interface parameters (io.Reader)
//   - Context-only methods
//   - Multiple non-error returns
type Store interface {
	// Get retrieves an item by ID.
	Get(ctx context.Context, id ID) (Item, error)

	// Put stores an item.
	Put(ctx context.Context, item Item) error

	// Delete removes an item by ID.
	Delete(ctx context.Context, id ID) error

	// List returns a paginated list of items.
	List(ctx context.Context, opts ListOptions) (ListResult, error)

	// Count returns the number of items. No error return.
	Count(ctx context.Context) int

	// Find retrieves multiple items by IDs. Variadic string-like parameter.
	Find(ctx context.Context, ids ...ID) ([]Item, error)

	// PutMany stores multiple items. Variadic struct parameter.
	PutMany(ctx context.Context, items ...Item) error

	// Import reads items from a reader. Interface parameter.
	Import(ctx context.Context, r io.Reader) (int, error)

	// Export writes items to a writer. Interface parameter + error return.
	Export(ctx context.Context, w io.Writer) error

	// GetOptional returns a pointer — nil means not found, no error.
	GetOptional(ctx context.Context, id ID) *Item

	// Touch updates the timestamp, returns the old and new values.
	// Multiple non-error returns.
	Touch(ctx context.Context, id ID) (before time.Time, after time.Time, err error)

	// Ping checks connectivity. Context-only, error-only.
	Ping(ctx context.Context) error

	// Tags returns all distinct tag values. Slice return, no error.
	Tags(ctx context.Context) []string

	// MetadataFor returns metadata for an item. Map return.
	MetadataFor(ctx context.Context, id ID) (map[string]string, error)

	// Close releases resources. No context, just error.
	Close() error
}
