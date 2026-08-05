// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package conformance is the corpus every testkit generator reads, plus the
// gate that proves the corpus is complete and the generated output is worth
// having.
//
// It is not published. Unlike a testdata tree these are ordinary packages, so
// `go test ./...` builds the generated code and runs the generated tests —
// which is the whole point: a corpus that only proves output was emitted
// proves nothing about whether the output tests anything.
//
// # Layout
//
// The corpus is indexed by the kind of declaration a generator reads, not by
// which generator reads it — four generators read interface methods, so an
// index by generator would duplicate every interface fixture four times.
//
//   - corpus/iface — interface methods, for stub, suite, bench, and model.
//     Subdivided by classification axis: detector (signature-driven),
//     contract (multi-role protocols), mixin (directive-driven, with the
//     negated form alongside the positive one), lang (Go type-system
//     variation), and composite (axes that modify each other's output).
//   - corpus/struct — struct fields, for builder.
//   - corpus/enum — typed constant blocks, for enum.
//   - corpus/errors — error variables, for sentinel. Note that sentinel opts
//     in at package scope, so these carry a package directive rather than a
//     var-level one.
//   - invalid — fixtures that must be rejected, for the semantic validator
//     behaviour a directive schema cannot express.
//
// Directories for unbuilt generators are absent rather than empty: an
// unregistered plugin declares no directives, so the gate demands no fixtures
// for it and starts demanding them the day it registers.
//
// # The gate
//
// Coverage is measured, never declared. Nothing in this module lists what it
// covers, because a list drifts and a fixture whose directive is misspelled
// would pass a list check on its directory name alone.
//
// Instead the gate annotates the corpus with eidos's own shape plugin,
// collects the classifications actually stamped, and diffs that against the
// registries. A misspelled directive stamps nothing and reports as a gap,
// which is what it is.
//
// # File naming
//
// Every fixture package holds exactly one `iface.go` carrying the declaration
// a generator reads, plus the types it needs. One name across the corpus means
// a reader opening any fixture knows where to look.
//
// # Hazards
//
// Fixtures are declarations only — no implementations. That is deliberate and
// temporary: a generated suite needs a subject to run against, and a subject
// with state baked into its constructor is untestable. A suite draws its own
// keys, so a pre-seeded entry is never the one drawn; and differential testing
// diverges on the first read, because the reference implementation starts
// empty and the subject does not.
//
// Whatever supplies subjects has to let the caller establish state rather than
// assuming it, and that mechanism is unresolved. Until it is, the corpus
// carries no implementations rather than carrying misleading ones.
package conformance
