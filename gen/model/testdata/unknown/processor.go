// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package unknown exercises the Unknown-shape fallback where a method
// can't be classified, verifying the "Skipped:" coverage header slot.
package unknown

import (
	"context"
	"errors"
)

//go:generate testkit model -o processortest/processor_model.gen.go Processor

// ErrNotFound is returned when an item is not found.
var ErrNotFound = errors.New("not found")

// Item is a stored value.
type Item struct {
	ID   string
	Name string
}

// Processor has a standard CRUD core plus an Unknown-shaped method.
type Processor interface {
	//testkit:errors ErrNotFound
	Get(ctx context.Context, id string) (Item, error)

	Put(ctx context.Context, item Item) error

	// Process has an Unknown shape: (ctx, string, int) (string, bool, error).
	// The generator can't classify it — it should appear in Skipped.
	Process(ctx context.Context, input string, count int) (string, bool, error)
}
