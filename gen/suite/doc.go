// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package suite implements the conformance suite generator for testkit.
//
// The generator reads Go interfaces annotated with //testkit: directives
// and produces Assert<Interface>Contract test harnesses with auto-detected
// subtests and typed plug-in extension points.
//
// # Architecture
//
// The generator follows a four-stage pipeline:
//
//	Analyze → Enrich → Validate → Render
//
// Analyze loads the Go package and extracts interface methods.
// Enrich applies directive-driven enrichers to add contract subtests.
// Validate checks for conflicting or invalid directive combinations.
// Render executes Go templates to produce the output file.
//
// # Method shape detection
//
// Each interface method is classified into a shape by [gen.DetectShape].
// The shape determines which typed context (ReaderContext, WriterContext, etc.)
// is used for plug-in assertions, and which auto-detected subtests are emitted.
// See the gen package documentation for the full detection rules and worked
// examples.
//
// # Generated output
//
// For an interface named Store, the generator produces:
//
//   - AssertStoreContract(t, factory, opts...) — entry point
//   - runStore<Method>(t, factory, cfg) — per-method subtests
//   - StorePrePopulate, StoreOn<Method>, StoreOnAll, StoreCustom — options
//   - storeConfig — internal option accumulator
//
// All option functions are prefixed with the interface name to prevent
// symbol collisions when multiple interfaces generate into the same
// output package.
package suite
