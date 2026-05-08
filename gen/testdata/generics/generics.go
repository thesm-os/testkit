// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package generics exercises generic type parameter propagation
// across all generators (suite, bench, stub, builder) and the gen
// package's own type analysis (loader, method, shape detection).
package generics

import (
	"context"
	"errors"
	"iter"
)

//go:generate testkit suite   -o cachetest/cache_spec.gen.go       Cache
//go:generate testkit bench   -o cachetest/cache_bench.gen.go      Cache
//go:generate testkit stub    -o cachetest/cache_stub.gen.go       Cache
//go:generate testkit builder -o cachetest/pair_builder.gen.go     Pair
//go:generate testkit builder -o cachetest/container_builder.gen.go Container

// ErrNotFound is returned when a key is not in the cache.
var ErrNotFound = errors.New("not found")

// --- Generic interface (exercises suite/bench/stub type param propagation) ---

// Cache is a generic key-value store. Type parameters propagate through
// all generated functions, types, options, and plug-in dispatch closures.
type Cache[K comparable, V any] interface {
	// Reader: func(ctx, K) (V, error).
	//testkit:errors ErrNotFound
	Get(ctx context.Context, key K) (V, error)

	// Writer: func(ctx, V) error.
	Put(ctx context.Context, val V) error

	// Deleter: func(ctx, K) error.
	//testkit:deleter
	Delete(ctx context.Context, key K) error

	// Aggregator: func(ctx) (T, error).
	Count(ctx context.Context) (int, error)

	// Pure: func() T.
	Len() int

	// StreamReader: func(ctx) iter.Seq2[V, error].
	Scan(ctx context.Context) iter.Seq2[V, error]

	// ReaderWithBool: func(ctx, K) (V, bool).
	Load(ctx context.Context, key K) (V, bool)
}

// --- Generic structs (exercises builder type param propagation) ---

// Pair holds two values of different types.
type Pair[A, B any] struct {
	First  A
	Second B
}

// Container holds items of a single type.
type Container[T any] struct {
	Label string
	Items []T
	Limit int
}

// Result wraps a value with an error — generic struct with interface field.
type Result[V any] struct {
	Value V
	Err   error
}

// Entry is a key-value pair used as a map entry builder target.
type Entry[K comparable, V any] struct {
	Key   K
	Value V
}
