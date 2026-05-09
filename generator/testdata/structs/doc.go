// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package structs holds struct-shape variations beyond the canonical
// Counter in basic. Generators that consume struct types (builder
// today; potentially more later) pick types from here by name; new
// edge cases land as additional types in this package, not as new
// sibling packages.
//
// Expected files (added as generators consume them):
//
//   - fields.go  — Item: every basic field shape (scalar, slice,
//                  map, []byte, with one unexported field for
//                  exported-only filtering)
//   - nested.go  — Order with embedded Metadata, nested Address
//                  values, and *Customer pointer field
//
// Type names address one concern each so a test can scope to its
// shape via `builder Item`, `builder Order`, etc. Each source file
// owns its own go:generate directive — the builder emits into a
// sibling structstest/ package using the source basename, so a
// listing groups source + generated impl + test by basename:
//
//   structs/fields.go      structs/structstest/fields.gen.go
//                          structs/structstest/fields.gen_test.go
package structs
