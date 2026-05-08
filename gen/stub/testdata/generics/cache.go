// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package generics exercises generic interface support for the
// stub generator.
package generics

import (
	"context"
	"errors"
)

//go:generate testkit stub -o cachetest/cache_stub.gen.go Cache

// ErrNotFound is returned when a key is not in the cache.
var ErrNotFound = errors.New("not found")

// Cache is a generic key-value cache with type parameters.
type Cache[K comparable, V any] interface {
	// Get is Reader-shaped.
	//testkit:errors ErrNotFound
	Get(ctx context.Context, key K) (V, error)

	// Put is Writer-shaped.
	Put(ctx context.Context, val V) error

	// Delete is Deleter-shaped.
	//testkit:deleter
	Delete(ctx context.Context, key K) error

	// Len is Pure-shaped.
	Len() int
}
