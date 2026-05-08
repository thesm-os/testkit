// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package generics holds type-parameterized fixtures. Multiple
// generators consume from here:
//
//   - builder generates fluent builders for generic structs
//     (Container[T], Pair[A,B]).
//   - stub will generate stubs for generic interfaces
//     (Cache[K, V]).
//   - suite/bench likewise reuse these types for their respective
//     test surfaces.
//
// Naming convention: each file names the type-parameter pattern it
// exercises (single-param, two-param, constrained, embedded). Tests
// scope by type name.
package generics
