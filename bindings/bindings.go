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

// MutatorBindings holds the reusable shape wiring for a Mutator-shaped method.
// A Mutator has the signature func(ctx, V) with no return value.
type MutatorBindings[T, V any] struct {
	Factory func() T
	Call    func(context.Context, T, V)
}

// ReaderWithBoolBindings holds the reusable shape wiring for a
// ReaderWithBool-shaped method: func(ctx, K) (V, bool) or func(K) (V, bool).
type ReaderWithBoolBindings[T any, K comparable, V any] struct {
	Factory func() T
	Call    func(context.Context, T, K) (V, bool)
}

// LookupBindings holds the reusable shape wiring for a Lookup-shaped method:
// func(ctx, K) (R1, R2, bool) or func(K) (R1, R2, bool).
type LookupBindings[T any, K comparable, V, R any] struct {
	Factory func() T
	Call    func(context.Context, T, K) (V, R, bool)
}

// PoisonAccessorBindings holds the reusable shape wiring for a
// PoisonAccessor-shaped method: func() error.
type PoisonAccessorBindings[T any] struct {
	Factory func() T
	Call    func(T) error
}

// ReaderNoErrorBindings holds the reusable shape wiring for a
// ReaderNoError-shaped method: func(ctx?, K) V — infallible
// lookups against in-memory state (caches, gauges, stable mappings).
// The Call adapter normalizes the "ctx? K → V" signature to a
// uniform shape regardless of whether the method takes context.
type ReaderNoErrorBindings[T any, K comparable, V any] struct {
	Factory func() T
	Call    func(context.Context, T, K) V
}

// PointerReaderBindings holds the reusable shape wiring for a
// PointerReader-shaped method: func(ctx?, K) *V — the nil-on-miss
// idiom. The Call adapter returns the raw pointer; downstream
// primitives compare against nil to detect misses.
type PointerReaderBindings[T any, K comparable, V any] struct {
	Factory func() T
	Call    func(context.Context, T, K) *V
}

// MultiReaderBindings holds the reusable shape wiring for a
// MultiReader-shaped method: func(ctx?, K) (V1, V2, error) —
// "get the entity + metadata" idioms.
type MultiReaderBindings[T any, K comparable, V1, V2 any] struct {
	Factory func() T
	Call    func(context.Context, T, K) (V1, V2, error)
}

// BatchReaderBindings holds the reusable shape wiring for a
// BatchReader-shaped method: func(ctx?, ...K) ([]V, error). The
// Call adapter passes the variadic key slice as a single []K.
type BatchReaderBindings[T any, K comparable, V any] struct {
	Factory func() T
	Call    func(context.Context, T, []K) ([]V, error)
}

// CompositeWriterBindings holds the reusable shape wiring for a
// CompositeWriter-shaped method: func(ctx?, K1, V) error —
// namespaced stores, tagged caches, two-key indexes.
type CompositeWriterBindings[T any, K1 comparable, V any] struct {
	Factory func() T
	Call    func(context.Context, T, K1, V) error
}

// MultiArgWriterBindings holds the reusable shape wiring for a
// MultiArgWriter-shaped method with 3 non-ctx params: func(ctx,
// p1, p2, p3) error. Used by the bench generator's typed hot-path
// measurements where boxing through `any` would defeat the
// allocation/latency contract.
type MultiArgWriterBindings[T any, P1, P2, P3 any] struct {
	Factory func() T
	Call    func(context.Context, T, P1, P2, P3) error
}

// MultiArgWriterVariadicBindings holds arity-agnostic shape wiring
// for a MultiArgWriter-shaped method. Suite uses this so contract
// baselines exercise methods with any non-ctx arity (2, 3, 4, …)
// from a single runtime; the generator emits a typed wrapper that
// converts `[]any` back to typed args at the call site, restoring
// type safety at the consumer boundary.
//
// Bench keeps the typed [MultiArgWriterBindings] above — boxing
// through `any` defeats hot-path measurements.
type MultiArgWriterVariadicBindings[T any] struct {
	Factory func() T
	Call    func(ctx context.Context, impl T, args ...any) error
}

// MultiAggregatorBindings holds the reusable shape wiring for a
// MultiAggregator-shaped method: func(ctx?) (V1, V2, error) —
// 2-tuple aggregations like Stats(ctx) (count, total, error).
type MultiAggregatorBindings[T any, V1, V2 any] struct {
	Factory func() T
	Call    func(context.Context, T) (V1, V2, error)
}

// VoidLifecycleBindings holds the reusable shape wiring for a
// VoidLifecycle-shaped method: func() or func(ctx) — Reset,
// parameterless lifecycle hooks. The Call adapter accepts ctx
// uniformly; impls without ctx parameters ignore the argument.
type VoidLifecycleBindings[T any] struct {
	Factory func() T
	Call    func(context.Context, T)
}

// StreamConsumerBindings holds the reusable shape wiring for a
// StreamConsumer-shaped method: func(ctx, S) (V, error) where S
// is interface-typed. The Call adapter accepts the interface-typed
// stream argument as `any`; downstream primitives type-assert as
// needed.
type StreamConsumerBindings[T, S, V any] struct {
	Factory func() T
	Call    func(context.Context, T, S) (V, error)
}
