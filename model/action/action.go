// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package action provides shape-typed Action helpers for model-based
// testing. Each helper eliminates the per-method boilerplate of
// drawing a sample, calling both SUT and reference, and comparing
// results. The generator emits one call per detected method.
package action

import (
	"context"
	"fmt"
	"sort"

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

// ReaderWithBool creates an action for a ReaderWithBool-shaped method:
// func(ctx, K) (V, bool) or func(K) (V, bool). Draws a key, calls both
// SUT and ref, compares value and ok flag.
func ReaderWithBool[T any, K comparable, V any](
	name string,
	keys *rapid.Generator[K],
	read func(context.Context, T, K) (V, bool),
) model.Action[T] {
	return model.Action[T]{
		Name: name,
		Run: func(rt *rapid.T, sut, ref T) {
			k := keys.Draw(rt, name+"_key")
			sutGot, sutOK := read(rt.Context(), sut, k)
			refGot, refOK := read(rt.Context(), ref, k)
			if sutOK != refOK {
				rt.Fatalf("%s(%v): SUT ok=%v, ref ok=%v", name, k, sutOK, refOK)
			}
			if sutOK {
				if diff := cmp.Diff(refGot, sutGot); diff != "" {
					rt.Fatalf("%s(%v): SUT/ref disagree:\n%s", name, k, diff)
				}
			}
		},
	}
}

// Lookup creates an action for a Lookup-shaped method:
// func(T, K) (R1, R2, bool). Draws a key, calls both SUT and ref,
// compares ok flag and both return values when present.
// The lookup closure should NOT take context — Lookup methods are
// typically pure reads. If the method takes ctx, the generated
// closure wraps it to pass context.Background().
// Lookup creates an action for a Lookup-shaped method:
// func(T, K) (R1, R2, bool). Draws a key, calls both SUT and ref,
// compares ok flag and R1 when present. R2 is compared via
// cmpOpts if provided (needed for uncomparable types like functions);
// otherwise skipped.
func Lookup[T any, K comparable, R1, R2 any](
	name string,
	keys *rapid.Generator[K],
	lookup func(T, K) (R1, R2, bool),
	cmpOpts ...cmp.Option,
) model.Action[T] {
	return model.Action[T]{
		Name: name,
		Run: func(rt *rapid.T, sut, ref T) {
			k := keys.Draw(rt, name+"_key")
			sutR1, sutR2, sutOK := lookup(sut, k)
			refR1, refR2, refOK := lookup(ref, k)
			if sutOK != refOK {
				rt.Fatalf("%s(%v): SUT ok=%v, ref ok=%v", name, k, sutOK, refOK)
			}
			if sutOK {
				if diff := cmp.Diff(refR1, sutR1); diff != "" {
					rt.Fatalf("%s(%v) R1: SUT/ref disagree:\n%s", name, k, diff)
				}
				if len(cmpOpts) > 0 {
					if diff := cmp.Diff(refR2, sutR2, cmpOpts...); diff != "" {
						rt.Fatalf("%s(%v) R2: SUT/ref disagree:\n%s", name, k, diff)
					}
				}
			}
		},
	}
}

// Mutator creates an action for a Mutator-shaped method: func(ctx, V)
// with no return. Calls both SUT and ref with the same drawn value.
// Divergence is detected by laws (not return-value comparison).
func Mutator[T, V any](
	name string,
	values *rapid.Generator[V],
	mutate func(context.Context, T, V),
) model.Action[T] {
	return model.Action[T]{
		Name: name,
		Run: func(rt *rapid.T, sut, ref T) {
			v := values.Draw(rt, name+"_value")
			mutate(rt.Context(), sut, v)
			mutate(rt.Context(), ref, v)
		},
	}
}

// PoisonCheck creates an action for a PoisonAccessor-shaped method:
// func() error. Calls both SUT and ref, compares error states. If the
// reference is poisoned, the SUT must also be poisoned (and vice versa).
func PoisonCheck[T any](
	name string,
	check func(T) error,
) model.Action[T] {
	return model.Action[T]{
		Name: name,
		Run: func(rt *rapid.T, sut, ref T) {
			sutErr := check(sut)
			refErr := check(ref)
			if (sutErr == nil) != (refErr == nil) {
				rt.Fatalf("%s: SUT err=%v, ref err=%v", name, sutErr, refErr)
			}
		},
	}
}

// Writer creates an action for a Writer-shaped method: func(ctx, V) error.
// Draws a value from the provided generator, calls both SUT and ref.
func Writer[T, V any](
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
func Pure[T, R any](
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
// compares. Order-insensitive: results are sorted by string
// representation before comparison since map-backed stores produce
// non-deterministic iteration order.
func Stream[T, V any](
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
				sortByString(sutItems)
				sortByString(refItems)
				if diff := cmp.Diff(refItems, sutItems); diff != "" {
					rt.Fatalf("%s: SUT/ref disagree:\n%s", name, diff)
				}
			}
		},
	}
}

// sortByString sorts a slice by the Sprint representation of each element.
func sortByString[V any](s []V) {
	sort.Slice(s, func(i, j int) bool {
		return fmt.Sprint(s[i]) < fmt.Sprint(s[j])
	})
}

// Stress creates an action that calls the SUT without comparing
// against the reference. Used for concurrent StressActions where
// only race detection matters — the SUT is mutated by concurrent
// linearizability workers, so comparison is meaningless.
func Stress[T any](
	name string,
	call func(T),
) model.Action[T] {
	return model.Action[T]{
		Name: name,
		Run: func(_ *rapid.T, sut, _ T) {
			call(sut)
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
