// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package basic is the primary test fixture for the testkit generator
// suite. It exercises all 7 generators (suite, bench, stub, builder,
// sentinel, enum, model-deferred) from a single source package.
//
// The gen package's own tests (loader, shape detection, directive
// parsing, field analysis) also load from this package, so symbols
// must not be removed or renamed without updating those tests.
package basic

import (
	"context"
	"errors"
	"fmt"
	"time"
)

//go:generate testkit suite    -o storetest/store_spec.gen.go      Store
//go:generate testkit bench    -o storetest/store_bench.gen.go     Store
//go:generate testkit stub     -o storetest/store_stub.gen.go      Store
//go:generate testkit builder  -o storetest/item_builder.gen.go    Item
//go:generate testkit sentinel -o errors.gen_test.go
//go:generate testkit enum     -o status_enum.gen_test.go          Status

// --- Errors ---

// ErrNotFound is returned when an item is not found.
var ErrNotFound = errors.New("basic: not found")

// ErrConflict is returned on duplicate key.
var ErrConflict = errors.New("basic: conflict")

// internalVar is unexported — should not appear in ErrorVars.
var internalVar = errors.New("basic: internal")

// --- Enum ---

// Status represents an item status.
type Status int

const (
	StatusPending Status = iota
	StatusActive
	StatusClosed
)

// --- Interface ---

// Store manages items.
type Store interface {
	// Get retrieves an item by ID.
	//testkit:errors ErrNotFound
	Get(ctx context.Context, id string) (Item, error)

	// Put stores an item.
	//testkit:nilsafe
	Put(ctx context.Context, item Item) error

	// Delete removes an item by ID.
	//testkit:deleter
	Delete(ctx context.Context, id string) error

	// Find retrieves multiple items by IDs (variadic).
	Find(ctx context.Context, ids ...string) ([]Item, error)

	// Count returns the number of stored items.
	//testkit:bounded 0 1000
	Count(ctx context.Context) int

	// Ping checks connectivity.
	//testkit:timeout 5s
	Ping(ctx context.Context) error

	// LegacyPut is deprecated in favor of PutBatch.
	//testkit:deprecated PutBatch
	LegacyPut(ctx context.Context, item Item) error
}

// --- Error types (used by gen/loader_test.go) ---

// ValidationError is a custom error type with fields.
type ValidationError struct {
	Field   string
	Message string
}

// Error implements the error interface.
func (e *ValidationError) Error() string {
	return fmt.Sprintf("basic: validation: %s: %s", e.Field, e.Message)
}

// NotFoundError has a custom Is method for matching.
type NotFoundError struct {
	Entity string
}

// Error implements the error interface.
func (e *NotFoundError) Error() string {
	return fmt.Sprintf("basic: %s not found", e.Entity)
}

// Is implements custom error matching — any NotFoundError matches.
func (e *NotFoundError) Is(target error) bool {
	_, ok := target.(*NotFoundError)
	return ok
}

// WrappedError demonstrates an error type with Unwrap.
type WrappedError struct {
	Cause error
}

// Error implements the error interface.
func (e *WrappedError) Error() string {
	return fmt.Sprintf("basic: wrapped: %v", e.Cause)
}

// Unwrap returns the underlying error.
func (e *WrappedError) Unwrap() error {
	return e.Cause
}

// --- Structs (used by builder and gen tests) ---

// Address is a nested struct for builder testing.
type Address struct {
	Street string
	City   string
}

// Item is a stored value. Exercises all common Go field types for
// builder generation: string, int, float64, bool, slice, byte slice,
// map, time.Time, nested struct, pointer-to-struct, unexported field.
type Item struct {
	ID       string
	Name     string
	Count    int
	Score    float64
	Active   bool
	Tags     []string
	Data     []byte
	Metadata map[string]string
	Created  time.Time
	Billing  Address
	Shipping *Address
	hidden   int // unexported — no setter
}
