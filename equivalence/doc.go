// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package equivalence ships pluggable equivalence relations consumed
// by differential-rollout's response comparison and replay's
// tolerance configuration. One [Relation] interface, one [Chain]
// composer, twelve built-ins covering the canonical migration-grade
// comparison cases (timestamps, generated IDs, retry counts,
// order-invariant collections, error classes).
//
// The design layers on top of [github.com/google/go-cmp/cmp]: each
// Relation contributes [cmp.Option] values that the Chain composes
// into one comparison run. go-cmp's FilterPath mechanism handles
// the "this relation applies to these paths only" routing
// natively — the Chain doesn't reimplement applies-vs-not-applies
// logic.
//
// Consumer-defined relations register through [Custom]; named
// chains are stored in a package-level preset registry. The
// testkit/registry/ package wraps the preset registry so plug-in
// modules can ship presets via blank-import.
package equivalence
