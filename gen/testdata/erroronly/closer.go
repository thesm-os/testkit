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

//go:generate testkit suite    -o closertest/closer_spec.gen.go  Closer
//go:generate testkit bench    -o closertest/closer_bench.gen.go Closer
//go:generate testkit stub     -o closertest/closer_stub.gen.go  Closer
//go:generate testkit sentinel -o errors.gen_test.go

// ErrClosed is returned when operating on a closed resource.
var ErrClosed = errors.New("erroronly: already closed")

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
