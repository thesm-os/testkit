// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package generics

//go:generate testkit stub  -o genericstest/holder.gen.go      Holder
//go:generate testkit stub  -o genericstest/keymap.gen.go      KeyMap
//go:generate testkit stub  -o genericstest/tally.gen.go       Tally
//go:generate testkit suite -o genericstest/holder_spec.gen_test.go Holder
//go:generate testkit suite -o genericstest/keymap_spec.gen_test.go KeyMap
//go:generate testkit suite -o genericstest/tally_spec.gen_test.go  Tally

import (
	"context"
	"errors"
)

// ErrNotFound is the sentinel for the generic interface fixtures.
var ErrNotFound = errors.New("generics: not found")

// Holder is a single-type-parameter generic interface — exercises
// the simplest generic stub case (one parameterized return type).
//
//testkit:idempotent
type Holder[V any] interface {
	// Get returns the value for the key, or ErrNotFound on miss.
	//
	//testkit:errors ErrNotFound
	Get(ctx context.Context, key string) (V, error)

	// Put stores the value for the key.
	Put(ctx context.Context, key string, value V) error

	// Delete removes the key.
	//
	//testkit:deleter
	Delete(ctx context.Context, key string) error
}

// KeyMap is a two-type-parameter generic interface — exercises
// generic stubs with multiple type parameters threading through
// every method.
type KeyMap[K comparable, V any] interface {
	// Get returns the value for the key, or ErrNotFound on miss.
	//
	//testkit:errors ErrNotFound
	Get(ctx context.Context, key K) (V, error)

	// Set stores the value for the key.
	Set(ctx context.Context, key K, value V) error
}

// Tally is a constrained-type-parameter generic interface.
// [Numeric] (declared in constraint.go alongside the builder
// fixtures) requires `~int | ~int64 | ~float64` — exercises
// constraint-aware concrete-type selection in the generic stub
// auto-test.
type Tally[T Numeric] interface {
	// Add increments the counter for the key by delta.
	Add(ctx context.Context, key string, delta T) error

	// Total returns the running total across all keys.
	Total(ctx context.Context) (T, error)
}
