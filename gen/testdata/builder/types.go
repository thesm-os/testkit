// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package builder

import (
	"context"
	"io"
	"time"
)

// ID is a named string type.
type ID string

// Status is a named integer type (enum pattern).
type Status int

const (
	StatusPending Status = iota
	StatusActive
	StatusClosed
)

// Address is a simple nested struct.
type Address struct {
	Street string
	City   string
	Zip    string
}

// Account exercises every field type category a builder generator must handle.
type Account struct {
	// Basic types
	Name    string
	Age     int
	Balance float64
	Active  bool

	// Named types
	AccountID ID
	Status    Status

	// Pointer types — nil pointer risk
	Nickname *string
	Manager  *Account
	Score    *int

	// Nested struct (value, not pointer)
	Address Address

	// Stdlib types
	CreatedAt time.Time
	TTL       time.Duration

	// Slice types
	Tags     []string
	Scores   []int
	Children []*Account

	// Map types
	Metadata map[string]string
	Counts   map[string]int

	// Array
	Checksum [32]byte

	// Interface fields
	Logger io.Writer

	// Function field
	OnChange func(context.Context) error

	// Channel
	Events chan string

	// Empty struct
	Marker struct{}

	// Unexported — must be skipped
	internal string
}

// Minimal is a struct with only one field — edge case for templates.
type Minimal struct {
	Value string
}

// Empty is a struct with no exported fields — edge case.
type Empty struct {
	hidden int
}
