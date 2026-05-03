// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package richstruct exercises rapid.MakeCustom with a struct containing
// every common Go type: primitives, named types, nested structs,
// pointer-to-struct, slices, maps, []byte, and combinations.
package richstruct

import (
	"context"
	"errors"
)

//go:generate testkit model -o storetest/store_model.gen.go Store

// ErrNotFound is returned when a document is not found.
var ErrNotFound = errors.New("not found")

// Priority is a named int type.
type Priority int

// Tag is a named string type.
type Tag string

// Address is a nested struct.
type Address struct {
	Street  string
	City    string
	ZipCode string
	Country string
}

// GeoPoint is a nested struct with floats.
type GeoPoint struct {
	Lat float64
	Lng float64
}

// Metadata holds nested pointers and maps.
type Metadata struct {
	Labels map[string]string
	Score  float32
}

// Document exercises every common field type that rapid.Make supports.
type Document struct {
	// Key field — drawn from the key pool.
	ID string

	// Primitives.
	Title    string
	Active   bool
	Views    int
	Size     int64
	Rating   float64
	Priority Priority // named int
	MainTag  Tag      // named string

	// Byte slice (common in real structs).
	Checksum []byte

	// Slice of primitives.
	Tags []string

	// Slice of named type.
	AllTags []Tag

	// Map of primitives.
	Attributes map[string]string

	// Nested struct (value).
	Address Address

	// Nested struct (pointer).
	Location *GeoPoint

	// Nested struct with map field.
	Meta Metadata

	// Pointer to primitive.
	OptionalNote *string

	// Numeric variety.
	SmallCount uint8
	Flags      uint32
	TinyScore  float32
	Offset     int32
}

// Store is a CRUD interface over Document.
type Store interface {
	//testkit:errors ErrNotFound
	Get(ctx context.Context, id string) (Document, error)

	Put(ctx context.Context, doc Document) error

	Delete(ctx context.Context, id string) error

	Count(ctx context.Context) (int, error)
}
