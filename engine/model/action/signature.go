// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

//nolint:errorlint // Action errors are diagnostic (SUT vs ref comparison), not wrapped.
package action

import (
	"context"
	"fmt"

	"github.com/google/go-cmp/cmp"
	"pgregory.net/rapid"

	"go.thesmos.sh/testkit/engine/model"
)

// ReaderNoError creates an action for a ReaderNoError-shaped method:
// func(ctx?, K) V — single key in, single value out, no error.
// Draws a key, calls both SUT and ref, compares results.
func ReaderNoError[T any, K comparable, V any](
	name string,
	keys *rapid.Generator[K],
	read func(context.Context, T, K) V,
) model.Action[T] {
	return model.Action[T]{
		Name: name,
		Kind: model.FailureSemantic,
		Run: func(rt *rapid.T, sut, ref T) model.ActionResult {
			k := keys.Draw(rt, name+"_key")
			sutGot := read(rt.Context(), sut, k)
			refGot := read(rt.Context(), ref, k)
			if diff := cmp.Diff(refGot, sutGot); diff != "" {
				return model.ActionResult{
					Err:    fmt.Errorf("%s(%v): SUT/ref disagree:\n%s", name, k, diff),
					Input:  k,
					Output: sutGot,
				}
			}
			return model.ActionResult{Input: k, Output: sutGot}
		},
	}
}

// PointerReader creates an action for a PointerReader-shaped method:
// func(ctx?, K) *V. Draws a key, calls both SUT and ref, compares
// pointed-to values when both non-nil and asserts both are nil
// or both non-nil otherwise.
func PointerReader[T any, K comparable, V any](
	name string,
	keys *rapid.Generator[K],
	read func(context.Context, T, K) *V,
) model.Action[T] {
	return model.Action[T]{
		Name: name,
		Kind: model.FailureSemantic,
		Run: func(rt *rapid.T, sut, ref T) model.ActionResult {
			k := keys.Draw(rt, name+"_key")
			sutGot := read(rt.Context(), sut, k)
			refGot := read(rt.Context(), ref, k)
			if (sutGot == nil) != (refGot == nil) {
				return model.ActionResult{
					Err:    fmt.Errorf("%s(%v): SUT nil=%v, ref nil=%v", name, k, sutGot == nil, refGot == nil),
					Input:  k,
					Output: sutGot,
				}
			}
			if sutGot != nil {
				if diff := cmp.Diff(*refGot, *sutGot); diff != "" {
					return model.ActionResult{
						Err:    fmt.Errorf("%s(%v): SUT/ref disagree:\n%s", name, k, diff),
						Input:  k,
						Output: sutGot,
					}
				}
			}
			return model.ActionResult{Input: k, Output: sutGot}
		},
	}
}

// MultiReader creates an action for a MultiReader-shaped method:
// func(ctx?, K) (V1, V2, error). Draws a key, calls both SUT and
// ref, compares both non-error values when err is nil.
func MultiReader[T any, K comparable, V1, V2 any](
	name string,
	keys *rapid.Generator[K],
	read func(context.Context, T, K) (V1, V2, error),
) model.Action[T] {
	return model.Action[T]{
		Name: name,
		Kind: model.FailureSemantic,
		Run: func(rt *rapid.T, sut, ref T) model.ActionResult {
			k := keys.Draw(rt, name+"_key")
			sutV1, sutV2, sutErr := read(rt.Context(), sut, k)
			refV1, refV2, refErr := read(rt.Context(), ref, k)
			out := MultiReaderOutput{V1: sutV1, V2: sutV2}
			if (sutErr == nil) != (refErr == nil) {
				return model.ActionResult{
					Err:    fmt.Errorf("%s(%v): SUT err=%v, ref err=%v", name, k, sutErr, refErr),
					Input:  k,
					Output: out,
				}
			}
			if sutErr == nil {
				if diff := cmp.Diff(refV1, sutV1); diff != "" {
					return model.ActionResult{
						Err:    fmt.Errorf("%s(%v) V1: SUT/ref disagree:\n%s", name, k, diff),
						Input:  k,
						Output: out,
					}
				}
				if diff := cmp.Diff(refV2, sutV2); diff != "" {
					return model.ActionResult{
						Err:    fmt.Errorf("%s(%v) V2: SUT/ref disagree:\n%s", name, k, diff),
						Input:  k,
						Output: out,
					}
				}
			}
			return model.ActionResult{Input: k, Output: out}
		},
	}
}

// BatchReader creates an action for a BatchReader-shaped method:
// func(ctx?, ...K) ([]V, error). Draws a slice of keys (1..maxBatch),
// calls both SUT and ref, compares results.
func BatchReader[T any, K comparable, V any](
	name string,
	keys *rapid.Generator[K],
	read func(context.Context, T, ...K) ([]V, error),
) model.Action[T] {
	return model.Action[T]{
		Name: name,
		Kind: model.FailureSemantic,
		Run: func(rt *rapid.T, sut, ref T) model.ActionResult {
			batch := rapid.SliceOfN(keys, 1, 8).Draw(rt, name+"_batch")
			sutGot, sutErr := read(rt.Context(), sut, batch...)
			refGot, refErr := read(rt.Context(), ref, batch...)
			if (sutErr == nil) != (refErr == nil) {
				return model.ActionResult{
					Err:    fmt.Errorf("%s(%v): SUT err=%v, ref err=%v", name, batch, sutErr, refErr),
					Input:  batch,
					Output: sutGot,
				}
			}
			if sutErr == nil {
				if diff := cmp.Diff(refGot, sutGot); diff != "" {
					return model.ActionResult{
						Err:    fmt.Errorf("%s(%v): SUT/ref disagree:\n%s", name, batch, diff),
						Input:  batch,
						Output: sutGot,
					}
				}
			}
			return model.ActionResult{Input: batch, Output: sutGot}
		},
	}
}

// CompositeWriter creates an action for a CompositeWriter-shaped
// method: func(ctx?, K1, V) error — two-arg write where the first
// key partitions, the second carries the value.
func CompositeWriter[T any, K1 comparable, V any](
	name string,
	keys *rapid.Generator[K1],
	values *rapid.Generator[V],
	write func(context.Context, T, K1, V) error,
) model.Action[T] {
	return model.Action[T]{
		Name: name,
		Kind: model.FailureSemantic,
		Run: func(rt *rapid.T, sut, ref T) model.ActionResult {
			k := keys.Draw(rt, name+"_key")
			v := values.Draw(rt, name+"_value")
			sutErr := write(rt.Context(), sut, k, v)
			refErr := write(rt.Context(), ref, k, v)
			if (sutErr == nil) != (refErr == nil) {
				return model.ActionResult{
					Err:   fmt.Errorf("%s(%v, %v): SUT err=%v, ref err=%v", name, k, v, sutErr, refErr),
					Input: [2]any{k, v},
				}
			}
			return model.ActionResult{Input: [2]any{k, v}}
		},
	}
}

// MultiArgWriter creates an action for a MultiArgWriter-shaped
// method: func(ctx, p1, p2, p3, ...) error — 3+ non-ctx params.
// Consumers supply a single drawn-args generator that returns a
// slice of any-typed values, plus a call shim that forwards the
// slice into the typed method.
func MultiArgWriter[T any](
	name string,
	args *rapid.Generator[[]any],
	write func(context.Context, T, []any) error,
) model.Action[T] {
	return model.Action[T]{
		Name: name,
		Kind: model.FailureSemantic,
		Run: func(rt *rapid.T, sut, ref T) model.ActionResult {
			a := args.Draw(rt, name+"_args")
			sutErr := write(rt.Context(), sut, a)
			refErr := write(rt.Context(), ref, a)
			if (sutErr == nil) != (refErr == nil) {
				return model.ActionResult{
					Err:   fmt.Errorf("%s(%v): SUT err=%v, ref err=%v", name, a, sutErr, refErr),
					Input: a,
				}
			}
			return model.ActionResult{Input: a}
		},
	}
}

// MultiAggregator creates an action for a MultiAggregator-shaped
// method: func(ctx?) (V1, V2, error). Calls both SUT and ref,
// compares both non-error values when err is nil.
func MultiAggregator[T, V1, V2 any](
	name string,
	agg func(context.Context, T) (V1, V2, error),
) model.Action[T] {
	return model.Action[T]{
		Name: name,
		Kind: model.FailureSemantic,
		Run: func(rt *rapid.T, sut, ref T) model.ActionResult {
			sutV1, sutV2, sutErr := agg(rt.Context(), sut)
			refV1, refV2, refErr := agg(rt.Context(), ref)
			out := MultiAggregatorOutput{V1: sutV1, V2: sutV2}
			if (sutErr == nil) != (refErr == nil) {
				return model.ActionResult{
					Err:    fmt.Errorf("%s: SUT err=%v, ref err=%v", name, sutErr, refErr),
					Output: out,
				}
			}
			if sutErr == nil {
				if diff := cmp.Diff(refV1, sutV1); diff != "" {
					return model.ActionResult{
						Err:    fmt.Errorf("%s V1: SUT/ref disagree:\n%s", name, diff),
						Output: out,
					}
				}
				if diff := cmp.Diff(refV2, sutV2); diff != "" {
					return model.ActionResult{
						Err:    fmt.Errorf("%s V2: SUT/ref disagree:\n%s", name, diff),
						Output: out,
					}
				}
			}
			return model.ActionResult{Output: out}
		},
	}
}

// StreamConsumer creates an action for a StreamConsumer-shaped
// method: func(ctx, S) (V, error) where S is a fixed
// interface-typed stream the caller constructs and passes in.
// The stream factory is invoked twice (once for SUT, once for ref)
// so each side sees an independent reader at the same starting
// state — necessary for io.Reader-style consumers that exhaust
// their input.
func StreamConsumer[T, S, V any](
	name string,
	stream func() S,
	consume func(context.Context, T, S) (V, error),
) model.Action[T] {
	return model.Action[T]{
		Name: name,
		Kind: model.FailureSemantic,
		Run: func(rt *rapid.T, sut, ref T) model.ActionResult {
			sutGot, sutErr := consume(rt.Context(), sut, stream())
			refGot, refErr := consume(rt.Context(), ref, stream())
			if (sutErr == nil) != (refErr == nil) {
				return model.ActionResult{
					Err:    fmt.Errorf("%s: SUT err=%v, ref err=%v", name, sutErr, refErr),
					Output: sutGot,
				}
			}
			if sutErr == nil {
				if diff := cmp.Diff(refGot, sutGot); diff != "" {
					return model.ActionResult{
						Err:    fmt.Errorf("%s: SUT/ref disagree:\n%s", name, diff),
						Output: sutGot,
					}
				}
			}
			return model.ActionResult{Output: sutGot}
		},
	}
}

// VoidLifecycle creates an action for a VoidLifecycle-shaped method:
// func() or func(ctx) — no params, no return. Calls both SUT and
// ref; divergence is detected by laws (not return-value comparison).
func VoidLifecycle[T any](
	name string,
	call func(context.Context, T),
) model.Action[T] {
	return model.Action[T]{
		Name: name,
		Kind: model.FailureSemantic,
		Run: func(rt *rapid.T, sut, ref T) model.ActionResult {
			call(rt.Context(), sut)
			call(rt.Context(), ref)
			return model.ActionResult{}
		},
	}
}
