// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package allshapes exercises all 13 typed method shapes + Unknown
// in one interface, proving end-to-end plug-in dispatch and typed
// primitive wiring across suite, bench, and stub generators.
package allshapes

import (
	"context"
	"errors"
	"iter"
)

//go:generate testkit suite -o servicetest/service_spec.gen.go Service
//go:generate testkit bench -o servicetest/service_bench.gen.go Service
//go:generate testkit stub  -o servicetest/service_stub.gen.go  Service

// ErrNotFound is returned when an item is not found.
var ErrNotFound = errors.New("not found")

// Item is a stored value.
type Item struct {
	ID   string
	Name string
}

// Metadata holds auxiliary info returned by Lookup.
type Metadata struct {
	Version string
}

// Service exercises all 13 typed shapes + Unknown.
type Service interface {
	// Reader: func(ctx, K) (V, error).
	//testkit:errors ErrNotFound
	Get(ctx context.Context, id string) (Item, error)

	// Writer: func(ctx, V) error.
	Put(ctx context.Context, item Item) error

	// Deleter: func(ctx, K) error + directive.
	//testkit:deleter
	Delete(ctx context.Context, id string) error

	// Aggregator: func(ctx) (T, error).
	Count(ctx context.Context) (int, error)

	// Lifecycle: func(ctx) error.
	Close(ctx context.Context) error

	// Pure: func() T.
	Describe() string

	// Predicate: func() bool.
	IsEmpty() bool

	// StreamReader: func(ctx) iter.Seq2[V, error].
	List(ctx context.Context) iter.Seq2[Item, error]

	// Mutator: func(ctx, V) — void return + directive.
	//testkit:mutator
	Touch(ctx context.Context, id string)

	// ReaderWithBool: func(ctx, K) (V, bool).
	Load(ctx context.Context, id string) (Item, bool)

	// Lookup: func(K) (R1, R2, bool).
	Inspect(id string) (Item, Metadata, bool)

	// PoisonAccessor: func() error.
	Err() error

	// Unknown: multi-param, doesn't match any typed shape.
	Merge(ctx context.Context, id string, patch Item)
}
