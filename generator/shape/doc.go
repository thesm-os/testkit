// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package shape classifies Go method signatures into canonical
// "shapes" that downstream generators (stub, suite, bench, model)
// use to emit type-safe contract assertions, fault helpers, and
// benchmark primitives.
//
// # Architecture
//
// Detection is a registry of priority-ordered [Detector]s. The
// [Registry] walks detectors in descending priority order; the
// first detector that returns true claims the method. Adding a
// new shape is a new detector file plus one registry entry — no
// edit to existing detectors.
//
//	type Detector interface {
//	    Name() string
//	    Priority() int
//	    Detect(Signature) (Info, bool)
//	}
//
// Detectors receive a pre-computed [Signature] view so each detector
// stays focused on its own pattern. The view exposes:
//
//   - HasCtx, NonCtxParams, Variadic — parameter taxonomy
//   - HasError, NonErrResults, ErrIdx — return taxonomy
//   - Iter — pre-detected iter.Seq/Seq2 info
//   - Directives — //testkit: directives on the method
//
// # Shape catalog
//
// Twenty-one detectors plus an Unknown fallback:
//
//	Streaming:  StreamReader, StreamConsumer
//	Reading:    Reader, ReaderNoError, ReaderWithBool, Lookup,
//	            PointerReader, MultiReader, BatchReader
//	Writing:    Writer, CompositeWriter, Mutator, Deleter,
//	            MultiArgWriter
//	Stateless:  Pure, Predicate, PoisonAccessor
//	Aggregate:  Aggregator, MultiAggregator
//	Lifecycle:  Lifecycle, VoidLifecycle
//
// # Ctx is optional
//
// Reader, Writer, Deleter, Aggregator detectors accept signatures
// with or without ctx. PoisonAccessor remains the canonical no-ctx
// error-only shape; Lifecycle requires ctx to disambiguate from
// PoisonAccessor.
//
// # Auto-detected Mutator
//
// Mutator is detected from the signature `func(ctx?, V)` (void
// return) without requiring //testkit:mutator. The directive
// //testkit:not-mutator opts a method out of Mutator detection.
package shape
