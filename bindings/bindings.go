// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package bindings

import (
	"context"
	"iter"
)

// ReaderBindings holds the reusable shape wiring for a Reader-shaped method.
// Shared substrate for shape-driven generators.
type ReaderBindings[T any, K comparable, V any] struct {
	Factory func() T
	Call    func(context.Context, T, K) (V, error)
}

// WriterBindings holds the reusable shape wiring for a Writer-shaped method.
// Shared substrate for shape-driven generators.
type WriterBindings[T any, V any] struct {
	Factory func() T
	Call    func(context.Context, T, V) error
}

// DeleterBindings holds the reusable shape wiring for a Deleter-shaped method.
// Shared substrate for shape-driven generators.
type DeleterBindings[T any, K comparable] struct {
	Factory func() T
	Call    func(context.Context, T, K) error
}

// StreamBindings holds the reusable shape wiring for a StreamReader-shaped method.
// Shared substrate for shape-driven generators.
type StreamBindings[T any, V any] struct {
	Factory func() T
	Call    func(context.Context, T) iter.Seq2[V, error]
}

// AggregatorBindings holds the reusable shape wiring for an Aggregator-shaped method.
// Shared substrate for shape-driven generators.
type AggregatorBindings[T any, R any] struct {
	Factory func() T
	Call    func(context.Context, T) (R, error)
}

// LifecycleBindings holds the reusable shape wiring for a Lifecycle-shaped method.
// Shared substrate for shape-driven generators.
type LifecycleBindings[T any] struct {
	Factory func() T
	Call    func(context.Context, T) error
}

// CrossBindings holds the reusable shape wiring for cross-method assertions.
// Shared substrate for shape-driven generators.
type CrossBindings[T any] struct {
	Factory func() T
}

// PureBindings holds the reusable shape wiring for a Pure-shaped method.
// Shared substrate for shape-driven generators.
type PureBindings[T any, R any] struct {
	Factory func() T
	Call    func(T) R
}

// PredicateBindings holds the reusable shape wiring for a Predicate-shaped method.
// Shared substrate for shape-driven generators.
type PredicateBindings[T any] struct {
	Factory func() T
	Call    func(T) bool
}
