// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package shape

import "go.thesmos.sh/testkit/generator"

// Shape is the canonical classification of a method signature. Each
// constant corresponds to one [Detector] in this package.
type Shape int

// Shape constants. Order is documentation only; classification is
// driven by [Detector] priority, not enum order.
const (
	Unknown Shape = iota

	// Reading shapes.
	Reader         // func(ctx?, K) (V, error)
	ReaderNoError  // func(ctx?, K) V
	ReaderWithBool // func(ctx?, K) (V, bool)
	Lookup         // func(ctx?, K) (R1, R2, bool)
	PointerReader  // func(ctx?, K) *V
	MultiReader    // func(ctx?, K) (V1, V2, error)
	BatchReader    // func(ctx?, ...K) ([]V, error)

	// Writing shapes.
	Writer          // func(ctx?, V) error or func(ctx?, V) (R, error)
	CompositeWriter // func(ctx?, K1, V) error
	Mutator         // func(ctx?, V) — void; auto-detected
	Deleter         // func(ctx?, K) error + //testkit:deleter
	MultiArgWriter  // func(ctx, p1, p2, p3, ...) error — 3+ non-ctx params

	// Aggregate shapes.
	Aggregator      // func(ctx?) (T, error) or func(ctx?) T
	MultiAggregator // func(ctx?) (V1, V2, error)

	// Streaming shapes.
	StreamReader   // returns iter.Seq[V] or iter.Seq2[V, error]
	StreamConsumer // func(ctx, S) (V, error) where S is interface-typed

	// Stateless shapes.
	Pure           // func() T — no params, no ctx, no error
	Predicate      // func() bool — no params, returns bool only
	PoisonAccessor // func() error — no ctx, no params, error-only

	// Lifecycle shapes.
	Lifecycle     // func(ctx) error — no other params
	VoidLifecycle // func() — no params, no return
)

// String returns the canonical name of the shape, as used in spec
// headers and template branching.
func (s Shape) String() string {
	switch s {
	case Reader:
		return "Reader"
	case ReaderNoError:
		return "ReaderNoError"
	case ReaderWithBool:
		return "ReaderWithBool"
	case Lookup:
		return "Lookup"
	case PointerReader:
		return "PointerReader"
	case MultiReader:
		return "MultiReader"
	case BatchReader:
		return "BatchReader"
	case Writer:
		return "Writer"
	case CompositeWriter:
		return "CompositeWriter"
	case Mutator:
		return "Mutator"
	case Deleter:
		return "Deleter"
	case MultiArgWriter:
		return "MultiArgWriter"
	case Aggregator:
		return "Aggregator"
	case MultiAggregator:
		return "MultiAggregator"
	case StreamReader:
		return "StreamReader"
	case StreamConsumer:
		return "StreamConsumer"
	case Pure:
		return "Pure"
	case Predicate:
		return "Predicate"
	case PoisonAccessor:
		return "PoisonAccessor"
	case Lifecycle:
		return "Lifecycle"
	case VoidLifecycle:
		return "VoidLifecycle"
	default:
		return "Unknown"
	}
}

// Info is the detection result for one method. Detectors populate
// the fields relevant to their shape; unused fields are empty.
type Info struct {
	// Shape is the detected classification. Always set; defaults
	// to Unknown when no detector matches.
	Shape Shape

	// KeyType is the rendered type for the primary key/input.
	// Populated by Reader, ReaderNoError, ReaderWithBool, Lookup,
	// PointerReader, MultiReader, BatchReader, CompositeWriter,
	// Deleter, StreamConsumer.
	KeyType string

	// KeyType2 is the second key for composite-key shapes.
	// Populated by CompositeWriter only.
	KeyType2 string

	// ValType is the rendered type for the primary value/output.
	// Populated by every reading shape, Writer, Mutator,
	// Aggregator, MultiAggregator, StreamConsumer.
	ValType string

	// ValType2 is the second value for multi-return shapes.
	// Populated by MultiReader, MultiAggregator.
	ValType2 string

	// RetType is the rendered type for the optional non-error
	// result of a Writer (Writer-with-result form).
	RetType string

	// Iter carries iter.Seq[T] / iter.Seq2[V, error] metadata for
	// StreamReader. Empty for other shapes.
	Iter generator.IterSeqInfo
}
