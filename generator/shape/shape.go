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

	// Contract-tier shapes. Promoted from a signature-tier base when
	// the interface structure (sibling methods, fields, directive
	// payload) supports the contract. Each carries invariants beyond
	// what the base shape implies.
	Persister       // Writer-with-result whose key is fetched by a sibling Reader
	Updater         // Writer/CompositeWriter that replaces by key
	Upserter        // idempotent insert-or-update
	CompareAndSwap  // Writer with version field; exactly-one-winner
	Appender        // Writer that returns monotonic offsets
	Watcher         // subscribes to changes triggered by a sibling
	Paginator       // paginated reader with cursor field
	GetOrCompute    // Reader with func(()V) arg, single-flight semantics
	TransactionFunc // Lifecycle/Writer taking func(Tx) error
	AcquireLease    // acquire paired with a release method
	Publisher       // publishes to a sibling subscribe method
	Subscriber      // channel-/callback-based subscriber

	// Composite-tier shapes. Multi-method shapes spanning ≥2
	// interface methods.
	Pool     // Get/Put pair
	Cursor   // Next/Close pair
	TwoPhase // Begin/Commit/Rollback triad
	Saga     // multi-step chain with compensation
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
	case Persister:
		return "Persister"
	case Updater:
		return "Updater"
	case Upserter:
		return "Upserter"
	case CompareAndSwap:
		return "CompareAndSwap"
	case Appender:
		return "Appender"
	case Watcher:
		return "Watcher"
	case Paginator:
		return "Paginator"
	case GetOrCompute:
		return "GetOrCompute"
	case TransactionFunc:
		return "TransactionFunc"
	case AcquireLease:
		return "AcquireLease"
	case Publisher:
		return "Publisher"
	case Subscriber:
		return "Subscriber"
	case Pool:
		return "Pool"
	case Cursor:
		return "Cursor"
	case TwoPhase:
		return "TwoPhase"
	case Saga:
		return "Saga"
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
