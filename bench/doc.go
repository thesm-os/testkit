// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package bench provides the runtime types and benchmark primitives
// for generated BenchmarkXxxContract harnesses.
//
// Each method shape has a typed Context (e.g. [ReaderContext]) and
// primitives that are either measurements ([ReaderHotPath],
// [ReaderConcurrentThroughput]) or deterministic gates
// ([ReaderAllocsWithin]).
//
// Parameterized shapes (Reader, Writer, Deleter, Stream) have HotPath
// primitives that vary the input. Parameterless shapes (Aggregator,
// Lifecycle, Pure, Predicate) rely on the auto-emitted default
// hot-path; plug-ins provide AllocsWithin gates only.
package bench
