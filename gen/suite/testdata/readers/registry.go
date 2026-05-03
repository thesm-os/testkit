// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package readers exercises the spec generator with Reader-shaped methods
// and plug-in primitives. Lookup is a non-CRUD Reader — proves the shape
// system works beyond CRUD store patterns.
package readers

import (
	"context"
	"errors"
	"iter"
)

//go:generate testkit suite -o registrytest/registry_spec.gen.go Registry

// ErrNotRegistered is returned when a handler is not found.
var ErrNotRegistered = errors.New("not registered")

// Handler is a registered handler.
type Handler struct {
	Name    string
	Version int
}

// Registry is a handler lookup service. Exercises Reader (Lookup),
// StreamReader (List), and Aggregator (Count) shapes.
type Registry interface {
	//testkit:errors ErrNotRegistered
	// Lookup retrieves a handler by name.
	Lookup(ctx context.Context, name string) (Handler, error)

	// List returns all registered handlers.
	List(ctx context.Context) iter.Seq2[Handler, error]

	// Count returns the number of registered handlers.
	Count(ctx context.Context) (int, error)
}
