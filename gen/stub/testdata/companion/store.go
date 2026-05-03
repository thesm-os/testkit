// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package companion demonstrates wrapping a hand-written in-memory
// implementation with the generated stub via DelegateTo. This is the
// load-bearing pattern for integration tests and sim companions:
// the real logic lives in the manual implementation, the stub adds
// recording, fault injection, and strict mode on top.
package companion

import (
	"context"
	"errors"
)

//go:generate testkit stub -o storetest/store_stub.gen.go Store

// ErrNotFound is returned when a key does not exist.
var ErrNotFound = errors.New("not found")

// Store is a simple key-value interface.
type Store interface {
	// Get retrieves a value by key.
	Get(ctx context.Context, key string) (string, error)
	// Put stores a value.
	Put(ctx context.Context, key string, value string) error
	// Delete removes a key.
	Delete(ctx context.Context, key string) error
}
