// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package testkit

import "context"

// Call types represent the input arguments to a shape's Call closure.
// These are the C parameter for Fault[C] when injecting faults into
// Bindings. Only pre-invocation inputs are captured — results are not
// available at fault-decision time.
//
// Ctx is stored as a struct field (not passed as a parameter) because
// these are value snapshots for fault-predicate inspection, not
// request-scoped flow. The containedctx lint is suppressed accordingly.

// ReaderCall captures the input arguments to a Reader-shaped Call.
// V is unused as a field but preserved for type-level distinction
// with ReaderBindings[T, K, V].
type ReaderCall[T any, K comparable, V any] struct {
	Ctx  context.Context //nolint:containedctx // fault predicate input, not request flow
	Impl T
	Key  K
}

// WriterCall captures the input arguments to a Writer-shaped Call.
type WriterCall[T any, V any] struct {
	Ctx  context.Context //nolint:containedctx // fault predicate input, not request flow
	Impl T
	Val  V
}

// DeleterCall captures the input arguments to a Deleter-shaped Call.
type DeleterCall[T any, K comparable] struct {
	Ctx  context.Context //nolint:containedctx // fault predicate input, not request flow
	Impl T
	Key  K
}

// AggregatorCall captures the input arguments to an Aggregator-shaped Call.
// R is unused as a field but preserved for type-level distinction
// with AggregatorBindings[T, R].
type AggregatorCall[T any, R any] struct {
	Ctx  context.Context //nolint:containedctx // fault predicate input, not request flow
	Impl T
}

// LifecycleCall captures the input arguments to a Lifecycle-shaped Call.
type LifecycleCall[T any] struct {
	Ctx  context.Context //nolint:containedctx // fault predicate input, not request flow
	Impl T
}

// WithReaderFaults wraps a ReaderBindings' Call to apply fault strategies.
// If no faults are provided, the bindings are returned unchanged (zero cost).
// Faults are evaluated in order; the first to fire short-circuits with its error.
func WithReaderFaults[T any, K comparable, V any](
	b ReaderBindings[T, K, V],
	clock Clock,
	faults ...Fault[ReaderCall[T, K, V]],
) ReaderBindings[T, K, V] {
	if len(faults) == 0 {
		return b
	}
	return ReaderBindings[T, K, V]{
		Factory: b.Factory,
		Call: func(ctx context.Context, impl T, k K) (V, error) {
			call := ReaderCall[T, K, V]{Ctx: ctx, Impl: impl, Key: k}
			for _, f := range faults {
				if fired, err := f.ShouldFire(call, clock); fired {
					var zero V
					return zero, err //nolint:wrapcheck // fault errors pass through unwrapped for errors.Is
				}
			}
			return b.Call(ctx, impl, k)
		},
	}
}

// WithWriterFaults wraps a WriterBindings' Call to apply fault strategies.
// If no faults are provided, the bindings are returned unchanged (zero cost).
// Faults are evaluated in order; the first to fire short-circuits with its error.
func WithWriterFaults[T, V any](
	b WriterBindings[T, V],
	clock Clock,
	faults ...Fault[WriterCall[T, V]],
) WriterBindings[T, V] {
	if len(faults) == 0 {
		return b
	}
	return WriterBindings[T, V]{
		Factory: b.Factory,
		Call: func(ctx context.Context, impl T, v V) error {
			call := WriterCall[T, V]{Ctx: ctx, Impl: impl, Val: v}
			for _, f := range faults {
				if fired, err := f.ShouldFire(call, clock); fired {
					return err //nolint:wrapcheck // fault errors pass through unwrapped for errors.Is
				}
			}
			return b.Call(ctx, impl, v)
		},
	}
}

// WithDeleterFaults wraps a DeleterBindings' Call to apply fault strategies.
// If no faults are provided, the bindings are returned unchanged (zero cost).
// Faults are evaluated in order; the first to fire short-circuits with its error.
func WithDeleterFaults[T any, K comparable](
	b DeleterBindings[T, K],
	clock Clock,
	faults ...Fault[DeleterCall[T, K]],
) DeleterBindings[T, K] {
	if len(faults) == 0 {
		return b
	}
	return DeleterBindings[T, K]{
		Factory: b.Factory,
		Call: func(ctx context.Context, impl T, k K) error {
			call := DeleterCall[T, K]{Ctx: ctx, Impl: impl, Key: k}
			for _, f := range faults {
				if fired, err := f.ShouldFire(call, clock); fired {
					return err //nolint:wrapcheck // fault errors pass through unwrapped for errors.Is
				}
			}
			return b.Call(ctx, impl, k)
		},
	}
}

// WithAggregatorFaults wraps an AggregatorBindings' Call to apply fault strategies.
// If no faults are provided, the bindings are returned unchanged (zero cost).
// Faults are evaluated in order; the first to fire short-circuits with its error.
func WithAggregatorFaults[T, R any](
	b AggregatorBindings[T, R],
	clock Clock,
	faults ...Fault[AggregatorCall[T, R]],
) AggregatorBindings[T, R] {
	if len(faults) == 0 {
		return b
	}
	return AggregatorBindings[T, R]{
		Factory: b.Factory,
		Call: func(ctx context.Context, impl T) (R, error) {
			call := AggregatorCall[T, R]{Ctx: ctx, Impl: impl}
			for _, f := range faults {
				if fired, err := f.ShouldFire(call, clock); fired {
					var zero R
					return zero, err //nolint:wrapcheck // fault errors pass through unwrapped for errors.Is
				}
			}
			return b.Call(ctx, impl)
		},
	}
}

// WithLifecycleFaults wraps a LifecycleBindings' Call to apply fault strategies.
// If no faults are provided, the bindings are returned unchanged (zero cost).
// Faults are evaluated in order; the first to fire short-circuits with its error.
func WithLifecycleFaults[T any](
	b LifecycleBindings[T],
	clock Clock,
	faults ...Fault[LifecycleCall[T]],
) LifecycleBindings[T] {
	if len(faults) == 0 {
		return b
	}
	return LifecycleBindings[T]{
		Factory: b.Factory,
		Call: func(ctx context.Context, impl T) error {
			call := LifecycleCall[T]{Ctx: ctx, Impl: impl}
			for _, f := range faults {
				if fired, err := f.ShouldFire(call, clock); fired {
					return err //nolint:wrapcheck // fault errors pass through unwrapped for errors.Is
				}
			}
			return b.Call(ctx, impl)
		},
	}
}
