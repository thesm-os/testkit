// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package suite

import (
	"context"
	"iter"
	"testing"

	"go.thesmos.sh/testkit/bindings"
)

// ReaderContextFor constructs a [ReaderContext] from a factory and a
// method-binding closure. Provided so generators emit one call instead
// of a 7-line struct literal per Reader-shape method.
func ReaderContextFor[T any, K comparable, V any](
	t *testing.T,
	factory func() T,
	call func(context.Context, T, K) (V, error),
) ReaderContext[T, K, V] {
	t.Helper()
	return ReaderContext[T, K, V]{
		T: t,
		ReaderBindings: bindings.ReaderBindings[T, K, V]{
			Factory: factory,
			Call:    call,
		},
	}
}

// ReaderNoErrorContextFor constructs a [ReaderNoErrorContext] from a
// factory and a method-binding closure.
func ReaderNoErrorContextFor[T any, K comparable, V any](
	t *testing.T,
	factory func() T,
	call func(context.Context, T, K) V,
) ReaderNoErrorContext[T, K, V] {
	t.Helper()
	return ReaderNoErrorContext[T, K, V]{
		T: t,
		ReaderNoErrorBindings: bindings.ReaderNoErrorBindings[T, K, V]{
			Factory: factory,
			Call:    call,
		},
	}
}

// ReaderWithBoolContextFor constructs a [ReaderWithBoolContext].
func ReaderWithBoolContextFor[T any, K comparable, V any](
	t *testing.T,
	factory func() T,
	call func(context.Context, T, K) (V, bool),
) ReaderWithBoolContext[T, K, V] {
	t.Helper()
	return ReaderWithBoolContext[T, K, V]{
		T: t,
		ReaderWithBoolBindings: bindings.ReaderWithBoolBindings[T, K, V]{
			Factory: factory,
			Call:    call,
		},
	}
}

// LookupContextFor constructs a [LookupContext].
func LookupContextFor[T any, K comparable, V, R any](
	t *testing.T,
	factory func() T,
	call func(context.Context, T, K) (V, R, bool),
) LookupContext[T, K, V, R] {
	t.Helper()
	return LookupContext[T, K, V, R]{
		T: t,
		LookupBindings: bindings.LookupBindings[T, K, V, R]{
			Factory: factory,
			Call:    call,
		},
	}
}

// PointerReaderContextFor constructs a [PointerReaderContext].
func PointerReaderContextFor[T any, K comparable, V any](
	t *testing.T,
	factory func() T,
	call func(context.Context, T, K) *V,
) PointerReaderContext[T, K, V] {
	t.Helper()
	return PointerReaderContext[T, K, V]{
		T: t,
		PointerReaderBindings: bindings.PointerReaderBindings[T, K, V]{
			Factory: factory,
			Call:    call,
		},
	}
}

// MultiReaderContextFor constructs a [MultiReaderContext].
func MultiReaderContextFor[T any, K comparable, V1, V2 any](
	t *testing.T,
	factory func() T,
	call func(context.Context, T, K) (V1, V2, error),
) MultiReaderContext[T, K, V1, V2] {
	t.Helper()
	return MultiReaderContext[T, K, V1, V2]{
		T: t,
		MultiReaderBindings: bindings.MultiReaderBindings[T, K, V1, V2]{
			Factory: factory,
			Call:    call,
		},
	}
}

// BatchReaderContextFor constructs a [BatchReaderContext].
func BatchReaderContextFor[T any, K comparable, V any](
	t *testing.T,
	factory func() T,
	call func(context.Context, T, []K) ([]V, error),
) BatchReaderContext[T, K, V] {
	t.Helper()
	return BatchReaderContext[T, K, V]{
		T: t,
		BatchReaderBindings: bindings.BatchReaderBindings[T, K, V]{
			Factory: factory,
			Call:    call,
		},
	}
}

// WriterContextFor constructs a [WriterContext].
func WriterContextFor[T, V any](
	t *testing.T,
	factory func() T,
	call func(context.Context, T, V) error,
) WriterContext[T, V] {
	t.Helper()
	return WriterContext[T, V]{
		T: t,
		WriterBindings: bindings.WriterBindings[T, V]{
			Factory: factory,
			Call:    call,
		},
	}
}

// CompositeWriterContextFor constructs a [CompositeWriterContext].
func CompositeWriterContextFor[T any, K1 comparable, V any](
	t *testing.T,
	factory func() T,
	call func(context.Context, T, K1, V) error,
) CompositeWriterContext[T, K1, V] {
	t.Helper()
	return CompositeWriterContext[T, K1, V]{
		T: t,
		CompositeWriterBindings: bindings.CompositeWriterBindings[T, K1, V]{
			Factory: factory,
			Call:    call,
		},
	}
}

// MutatorContextFor constructs a [MutatorContext].
func MutatorContextFor[T, V any](
	t *testing.T,
	factory func() T,
	call func(context.Context, T, V),
) MutatorContext[T, V] {
	t.Helper()
	return MutatorContext[T, V]{
		T: t,
		MutatorBindings: bindings.MutatorBindings[T, V]{
			Factory: factory,
			Call:    call,
		},
	}
}

// DeleterContextFor constructs a [DeleterContext].
func DeleterContextFor[T any, K comparable](
	t *testing.T,
	factory func() T,
	call func(context.Context, T, K) error,
) DeleterContext[T, K] {
	t.Helper()
	return DeleterContext[T, K]{
		T: t,
		DeleterBindings: bindings.DeleterBindings[T, K]{
			Factory: factory,
			Call:    call,
		},
	}
}

// MultiArgWriterContextFor constructs a [MultiArgWriterContext].
func MultiArgWriterContextFor[T, P1, P2, P3 any](
	t *testing.T,
	factory func() T,
	call func(context.Context, T, P1, P2, P3) error,
) MultiArgWriterContext[T, P1, P2, P3] {
	t.Helper()
	return MultiArgWriterContext[T, P1, P2, P3]{
		T: t,
		MultiArgWriterBindings: bindings.MultiArgWriterBindings[T, P1, P2, P3]{
			Factory: factory,
			Call:    call,
		},
	}
}

// AggregatorContextFor constructs an [AggregatorContext].
func AggregatorContextFor[T, R any](
	t *testing.T,
	factory func() T,
	call func(context.Context, T) (R, error),
) AggregatorContext[T, R] {
	t.Helper()
	return AggregatorContext[T, R]{
		T: t,
		AggregatorBindings: bindings.AggregatorBindings[T, R]{
			Factory: factory,
			Call:    call,
		},
	}
}

// MultiAggregatorContextFor constructs a [MultiAggregatorContext].
func MultiAggregatorContextFor[T, V1, V2 any](
	t *testing.T,
	factory func() T,
	call func(context.Context, T) (V1, V2, error),
) MultiAggregatorContext[T, V1, V2] {
	t.Helper()
	return MultiAggregatorContext[T, V1, V2]{
		T: t,
		MultiAggregatorBindings: bindings.MultiAggregatorBindings[T, V1, V2]{
			Factory: factory,
			Call:    call,
		},
	}
}

// StreamContextFor constructs a [StreamContext] for a StreamReader.
func StreamContextFor[T, V any](
	t *testing.T,
	factory func() T,
	call func(context.Context, T) iter.Seq2[V, error],
) StreamContext[T, V] {
	t.Helper()
	return StreamContext[T, V]{
		T: t,
		StreamBindings: bindings.StreamBindings[T, V]{
			Factory: factory,
			Call:    call,
		},
	}
}

// StreamConsumerContextFor constructs a [StreamConsumerContext].
func StreamConsumerContextFor[T, S, V any](
	t *testing.T,
	factory func() T,
	call func(context.Context, T, S) (V, error),
) StreamConsumerContext[T, S, V] {
	t.Helper()
	return StreamConsumerContext[T, S, V]{
		T: t,
		StreamConsumerBindings: bindings.StreamConsumerBindings[T, S, V]{
			Factory: factory,
			Call:    call,
		},
	}
}

// LifecycleContextFor constructs a [LifecycleContext].
func LifecycleContextFor[T any](
	t *testing.T,
	factory func() T,
	call func(context.Context, T) error,
) LifecycleContext[T] {
	t.Helper()
	return LifecycleContext[T]{
		T: t,
		LifecycleBindings: bindings.LifecycleBindings[T]{
			Factory: factory,
			Call:    call,
		},
	}
}

// VoidLifecycleContextFor constructs a [VoidLifecycleContext].
func VoidLifecycleContextFor[T any](
	t *testing.T,
	factory func() T,
	call func(context.Context, T),
) VoidLifecycleContext[T] {
	t.Helper()
	return VoidLifecycleContext[T]{
		T: t,
		VoidLifecycleBindings: bindings.VoidLifecycleBindings[T]{
			Factory: factory,
			Call:    call,
		},
	}
}

// PureContextFor constructs a [PureContext].
func PureContextFor[T, R any](
	t *testing.T,
	factory func() T,
	call func(T) R,
) PureContext[T, R] {
	t.Helper()
	return PureContext[T, R]{
		T: t,
		PureBindings: bindings.PureBindings[T, R]{
			Factory: factory,
			Call:    call,
		},
	}
}

// PredicateContextFor constructs a [PredicateContext].
func PredicateContextFor[T any](
	t *testing.T,
	factory func() T,
	call func(T) bool,
) PredicateContext[T] {
	t.Helper()
	return PredicateContext[T]{
		T: t,
		PredicateBindings: bindings.PredicateBindings[T]{
			Factory: factory,
			Call:    call,
		},
	}
}

// PoisonAccessorContextFor constructs a [PoisonAccessorContext].
func PoisonAccessorContextFor[T any](
	t *testing.T,
	factory func() T,
	call func(T) error,
) PoisonAccessorContext[T] {
	t.Helper()
	return PoisonAccessorContext[T]{
		T: t,
		PoisonAccessorBindings: bindings.PoisonAccessorBindings[T]{
			Factory: factory,
			Call:    call,
		},
	}
}
