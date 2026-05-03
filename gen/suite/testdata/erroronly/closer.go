// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package erroronly exercises the spec generator with methods that return
// only error — no non-error results. Tests that bounded and pure are not
// accidentally generated for void-like methods.
package erroronly

import (
	"context"
	"errors"
)

//go:generate testkit suite -o closertest/closer_spec.gen.go Closer

// ErrClosed is returned when operating on a closed resource.
var ErrClosed = errors.New("already closed")

// Closer is a resource lifecycle interface.
type Closer interface {
	//testkit:ctx
	//testkit:nilsafe
	// Open prepares the resource.
	Open(ctx context.Context) error

	//testkit:ctx
	//testkit:errors ErrClosed
	// Close releases the resource.
	Close(ctx context.Context) error
}
