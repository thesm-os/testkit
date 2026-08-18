// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// The *Of constructors: every shape whose consumer closure would only
// restate a method call, accepted as the interface method expression
// instead. Go spells Log.Append as func(Log, context.Context, Entry)
// error — receiver first — so each constructor here reorders that
// shape into the ctx-first one its sibling takes and delegates.
//
// They exist to remove a defect class, not keystrokes: a delegation
// closure is the one place an emitted action can call a method other
// than the one it claims and still compile. Naming the method once,
// as an expression, makes that mistake unspellable. A closure that
// does more than delegate — bind an argument, record history, drain a
// produced value — keeps using the closure-shaped sibling; these are
// for delegation only.

package action

import (
	"context"

	"pgregory.net/rapid"

	"go.thesmos.sh/testkit/engine/model"
)

// WriterOf is [Writer] for an interface method expression:
// func(T, ctx, V) error.
func WriterOf[T, V any](
	name string,
	values *rapid.Generator[V],
	m func(T, context.Context, V) error,
) model.Action[T] {
	return Writer(name, values, func(ctx context.Context, t T, v V) error {
		return m(t, ctx, v)
	})
}

// ReaderOf is [Reader] for an interface method expression:
// func(T, ctx, K) (V, error). Options pass through unchanged.
func ReaderOf[T any, K comparable, V any](
	name string,
	keys *rapid.Generator[K],
	m func(T, context.Context, K) (V, error),
	options ...Opt,
) model.Action[T] {
	return Reader(name, keys, func(ctx context.Context, t T, k K) (V, error) {
		return m(t, ctx, k)
	}, options...)
}

// AggregatorOf is [Aggregator] for an interface method expression:
// func(T, ctx) (R, error).
func AggregatorOf[T any, R comparable](
	name string,
	m func(T, context.Context) (R, error),
) model.Action[T] {
	return Aggregator(name, func(ctx context.Context, t T) (R, error) {
		return m(t, ctx)
	})
}

// DeleterOf is [Deleter] for an interface method expression:
// func(T, ctx, K) error.
func DeleterOf[T any, K comparable](
	name string,
	keys *rapid.Generator[K],
	m func(T, context.Context, K) error,
) model.Action[T] {
	return Deleter(name, keys, func(ctx context.Context, t T, k K) error {
		return m(t, ctx, k)
	})
}

// LifecycleOf is [Lifecycle] for an interface method expression:
// func(T, ctx) error.
func LifecycleOf[T any](
	name string,
	m func(T, context.Context) error,
) model.Action[T] {
	return Lifecycle(name, func(ctx context.Context, t T) error {
		return m(t, ctx)
	})
}

// EvictingReaderOf is [EvictingReader] for an interface method
// expression: func(T, ctx, K) (V, bool). The asymmetric comparison is
// the sibling's, unchanged.
func EvictingReaderOf[T any, K, V comparable](
	name string,
	keys *rapid.Generator[K],
	m func(T, context.Context, K) (V, bool),
) model.Action[T] {
	return EvictingReader(name, keys, func(ctx context.Context, t T, k K) (V, bool) {
		return m(t, ctx, k)
	})
}

// CompositeWriterOf is [CompositeWriter] for an interface method
// expression: func(T, ctx, K1, V) error.
func CompositeWriterOf[T any, K1 comparable, V any](
	name string,
	keys *rapid.Generator[K1],
	values *rapid.Generator[V],
	m func(T, context.Context, K1, V) error,
) model.Action[T] {
	return CompositeWriter(name, keys, values, func(ctx context.Context, t T, k K1, v V) error {
		return m(t, ctx, k, v)
	})
}

// PoolOf is [Pool] for a pair of interface method expressions:
// get func(T, ctx) (R, error), put func(T, ctx, R) error.
func PoolOf[T, R any](
	name string,
	get func(T, context.Context) (R, error),
	put func(T, context.Context, R) error,
) model.Action[T] {
	return Pool(name,
		func(ctx context.Context, t T) (R, error) { return get(t, ctx) },
		func(ctx context.Context, t T, r R) error { return put(t, ctx, r) },
	)
}

// StreamOf is [Stream] for an interface method expression:
// func(T, ctx) ([]V, error). A drain composition — open, walk, close —
// is not a method expression; it keeps the closure-shaped sibling.
func StreamOf[T, V any](
	name string,
	m func(T, context.Context) ([]V, error),
) model.Action[T] {
	return Stream(name, func(ctx context.Context, t T) ([]V, error) {
		return m(t, ctx)
	})
}
