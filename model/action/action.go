// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package action provides shape-typed Action helpers for model-based
// testing. Each helper eliminates the per-method boilerplate of
// drawing a sample, calling both SUT and reference, and comparing
// results. The generator emits one call per detected method.
package action

import (
	"context"

	"github.com/google/go-cmp/cmp"
	"pgregory.net/rapid"

	"go.thesmos.sh/testkit/model"
)

// Reader creates an action for a Reader-shaped method: func(ctx, K) (V, error).
// Draws a key from the provided generator, calls both SUT and ref, and
// compares results.
func Reader[T any, K comparable, V any](
	name string,
	keys *rapid.Generator[K],
	read func(context.Context, T, K) (V, error),
) model.Action[T] {
	return model.Action[T]{
		Name: name,
		Run: func(rt *rapid.T, sut, ref T) {
			k := keys.Draw(rt, name+"_key")
			sutGot, sutErr := read(rt.Context(), sut, k)
			refGot, refErr := read(rt.Context(), ref, k)
			if (sutErr == nil) != (refErr == nil) {
				rt.Fatalf("%s(%v): SUT err=%v, ref err=%v", name, k, sutErr, refErr)
			}
			if sutErr == nil {
				if diff := cmp.Diff(refGot, sutGot); diff != "" {
					rt.Fatalf("%s(%v): SUT/ref disagree:\n%s", name, k, diff)
				}
			}
		},
	}
}

// Writer creates an action for a Writer-shaped method: func(ctx, V) error.
// Draws a value from the provided generator, calls both SUT and ref.
func Writer[T any, V any](
	name string,
	values *rapid.Generator[V],
	write func(context.Context, T, V) error,
) model.Action[T] {
	return model.Action[T]{
		Name: name,
		Run: func(rt *rapid.T, sut, ref T) {
			v := values.Draw(rt, name+"_value")
			sutErr := write(rt.Context(), sut, v)
			refErr := write(rt.Context(), ref, v)
			if (sutErr == nil) != (refErr == nil) {
				rt.Fatalf("%s(%v): SUT err=%v, ref err=%v", name, v, sutErr, refErr)
			}
		},
	}
}

// Deleter creates an action for a Deleter-shaped method: func(ctx, K) error.
// Draws a key from the provided generator, calls both SUT and ref.
func Deleter[T any, K comparable](
	name string,
	keys *rapid.Generator[K],
	del func(context.Context, T, K) error,
) model.Action[T] {
	return model.Action[T]{
		Name: name,
		Run: func(rt *rapid.T, sut, ref T) {
			k := keys.Draw(rt, name+"_key")
			sutErr := del(rt.Context(), sut, k)
			refErr := del(rt.Context(), ref, k)
			if (sutErr == nil) != (refErr == nil) {
				rt.Fatalf("%s(%v): SUT err=%v, ref err=%v", name, k, sutErr, refErr)
			}
		},
	}
}

// Aggregator creates an action for an Aggregator-shaped method: func(ctx) (R, error).
// Calls both SUT and ref, compares results.
func Aggregator[T any, R comparable](
	name string,
	agg func(context.Context, T) (R, error),
) model.Action[T] {
	return model.Action[T]{
		Name: name,
		Run: func(rt *rapid.T, sut, ref T) {
			sutGot, sutErr := agg(rt.Context(), sut)
			refGot, refErr := agg(rt.Context(), ref)
			if (sutErr == nil) != (refErr == nil) {
				rt.Fatalf("%s: SUT err=%v, ref err=%v", name, sutErr, refErr)
			}
			if sutErr == nil && sutGot != refGot {
				rt.Fatalf("%s: SUT=%v, ref=%v", name, sutGot, refGot)
			}
		},
	}
}

// Lifecycle creates an action for a Lifecycle-shaped method: func(ctx) error.
// Calls both SUT and ref, compares error outcomes.
func Lifecycle[T any](
	name string,
	call func(context.Context, T) error,
) model.Action[T] {
	return model.Action[T]{
		Name: name,
		Run: func(rt *rapid.T, sut, ref T) {
			sutErr := call(rt.Context(), sut)
			refErr := call(rt.Context(), ref)
			if (sutErr == nil) != (refErr == nil) {
				rt.Fatalf("%s: SUT err=%v, ref err=%v", name, sutErr, refErr)
			}
		},
	}
}

// Pure creates an action for a Pure-shaped method: func(T) R.
// Calls both SUT and ref, compares results. No context.
func Pure[T any, R any](
	name string,
	call func(T) R,
) model.Action[T] {
	return model.Action[T]{
		Name: name,
		Run: func(rt *rapid.T, sut, ref T) {
			sutGot := call(sut)
			refGot := call(ref)
			if diff := cmp.Diff(refGot, sutGot); diff != "" {
				rt.Fatalf("%s: SUT/ref disagree:\n%s", name, diff)
			}
		},
	}
}

// Predicate creates an action for a Predicate-shaped method: func(T) bool.
// Calls both SUT and ref, compares results. No context.
func Predicate[T any](
	name string,
	call func(T) bool,
) model.Action[T] {
	return model.Action[T]{
		Name: name,
		Run: func(rt *rapid.T, sut, ref T) {
			sutGot := call(sut)
			refGot := call(ref)
			if sutGot != refGot {
				rt.Fatalf("%s: SUT=%v, ref=%v", name, sutGot, refGot)
			}
		},
	}
}

// Stream creates an action for a StreamReader-shaped method that
// returns all items. Calls both SUT and ref, collects results,
// compares. Uses the provided collect function to drain the iterator.
func Stream[T any, V any](
	name string,
	collect func(context.Context, T) ([]V, error),
) model.Action[T] {
	return model.Action[T]{
		Name: name,
		Run: func(rt *rapid.T, sut, ref T) {
			sutItems, sutErr := collect(rt.Context(), sut)
			refItems, refErr := collect(rt.Context(), ref)
			if (sutErr == nil) != (refErr == nil) {
				rt.Fatalf("%s: SUT err=%v, ref err=%v", name, sutErr, refErr)
			}
			if sutErr == nil {
				if diff := cmp.Diff(refItems, sutItems); diff != "" {
					rt.Fatalf("%s: SUT/ref disagree:\n%s", name, diff)
				}
			}
		},
	}
}

// Unknown creates an action for an Unknown-shaped method.
// Consumer provides the full comparison logic.
func Unknown[T any](
	name string,
	run func(rt *rapid.T, sut, ref T),
) model.Action[T] {
	return model.Action[T]{
		Name: name,
		Run: func(rt *rapid.T, sut, ref T) {
			run(rt, sut, ref)
		},
	}
}

