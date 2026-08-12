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
// Fixture packages declare interfaces; their subjects live beside the
// generated harness, in each `<pkg>test` directory's `inmemory.go`. The
// split is load-bearing both ways: a fixture package stating only a shape
// can be generated for and compiled, and the in-memory subject is what makes
// the corpus *run* the generated checks rather than merely emit them. Every
// subject starts empty and lets the harness establish state — the seed, the
// sequences, the laws all write before they read — because a subject with
// state baked into its constructor would collide with the keys the suite
// draws and diverge from a reference that starts empty.
package conformance
