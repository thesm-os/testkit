// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package allshapes exercises every method shape in one interface,
// proving end-to-end On<Method> dispatch + typed primitive wiring
// for Reader, Writer, Deleter, Aggregator, Lifecycle, Pure, Predicate,
// and StreamReader.
package allshapes

import (
	"context"
	"errors"
	"iter"
)

//go:generate testkit suite -o servicetest/service_spec.gen.go Service

// ErrNotFound is returned when an item is not found.
var ErrNotFound = errors.New("not found")

// Item is a stored value.
type Item struct {
	ID   string
	Name string
}

// Service exercises every method shape.
type Service interface {
	// Get is Reader-shaped: func(ctx, K) (V, error).
	//testkit:errors ErrNotFound
	Get(ctx context.Context, id string) (Item, error)

	// Put is Writer-shaped: func(ctx, V) error.
	Put(ctx context.Context, item Item) error

	// Delete is Deleter-shaped (with directive): func(ctx, K) error.
	//testkit:deleter
	Delete(ctx context.Context, id string) error

	// Count is Aggregator-shaped: func(ctx) (T, error).
	Count(ctx context.Context) (int, error)

	// Close is Lifecycle-shaped: func(ctx) error.
	Close(ctx context.Context) error

	// Describe is Pure-shaped: func() T (no ctx, no error).
	Describe() string

	// IsEmpty is Predicate-shaped: func() bool (no ctx).
	IsEmpty() bool

	// List is StreamReader-shaped: func(ctx) iter.Seq2[V, error].
	List(ctx context.Context) iter.Seq2[Item, error]
}
