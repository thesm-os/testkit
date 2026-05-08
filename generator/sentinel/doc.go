// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package sentinel implements the sentinel generator. It scans a
// package for two sources of error contracts and emits a test file
// that asserts their semantic invariants:
//
//  1. Exported package-level Err* variables — sentinels asserted for
//     prefix consistency, uniqueness across the set, non-overlap
//     under errors.Is, and unwrap-chain preservation.
//
//  2. Exported types implementing the error interface — round-tripped
//     through errors.As, asserted for format-string completeness,
//     and (when the type provides them) Is/Unwrap chain coverage.
//
// The generator scans the whole package by default; the
// [generator.Options.SourceFile] field narrows the Err* scan to one
// file (file-scoped via $GOFILE).
//
// # Architecture
//
// Per the rebuild's A5 decision, sentinel-specific scans live in this
// package — not on [generator.Package]. Callers use:
//
//	vars  := sentinel.ScanErrorVars(pkg, opts.SourceFile)
//	types := sentinel.ScanErrorTypes(pkg)
//	hasIs := sentinel.HasIsMethod(pkg, "ValidationError")
//
// The generator itself is a thin [generator.Pipeline] config: it
// returns an empty [generator.Result] when the package has no Err*
// vars and no error types, otherwise it runs the standard pipeline.
//
// # Cross-package non-overlap (G24)
//
// ANALYSIS.md G24 calls for opt-in checks that sentinels in this
// package don't accidentally match sentinels in another package via
// errors.Is. The directive `//testkit:sentinel-no-overlap-with`
// declares the additional packages to include in the matrix; the
// generator emits per-pair non-overlap subtests when present.
package sentinel
