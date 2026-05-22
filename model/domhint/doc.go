// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package domhint registers consumer-supplied
// [pgregory.net/rapid.Generator] values for opaque domain types
// (e.g., `ids.RunID`, `ids.FenceToken`) that the model generator
// cannot synthesize from reflection alone.
//
// The model generator's analysis detects opaque-typed parameters,
// looks them up in a [Registry], and emits a typed option
// constructor (`<Iface>ModelWith<Type>Gen`) when a hint is
// registered. Missing hints with non-reflection-generatable types
// error at codegen time, with a directive-guidance diagnostic
// (`add //testkit:domain-gen <Type> <Func>` or supply via the
// per-method option at test time).
//
// Two registration paths:
//
//   - At init time, via the package-level [Register] generic
//     function plus the [DefaultRegistry] singleton — the
//     `//testkit:domain-gen` directive's emitted code lands here.
//   - Per-test, via a fresh [Registry] threaded through the model
//     runner's options — useful for fixture-scoped hints.
//
// Collisions are the always-err policy: registering twice for the
// same type panics at registration time.
package domhint
