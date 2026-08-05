// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package linearize provides typed helpers for building Porcupine
// linearizability models. Three tiers:
//
//   - [KV] — pre-built for CRUD. Generator emits this.
//   - [NewModelBuilder] — typed builder for custom state.
//   - Raw porcupine.Model — escape hatch.
//
// Reader/Writer/Deleter/ReaderWithBool/Lookup operations get full
// Porcupine linearizability checking with P-compositionality
// (partitioned by key). Other shapes (Aggregator, Lifecycle, Pure,
// Predicate, Stream, Mutator, PoisonAccessor) run concurrently as
// stress actions but are not linearizability-checked.
package linearize
