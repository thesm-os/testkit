// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package builder generates a fluent builder for every struct annotated
// `//testkit:builder`.
//
// A test that constructs a value by composite literal restates every field
// each time, so a field added to the struct breaks every literal at once and a
// reader cannot tell which fields the test actually cares about. A builder
// inverts that: the constructor supplies the rest, and the test states only
// what it varies.
//
// # Directive
//
//	//testkit:builder
//	type Config struct { ... }
//
// `defaults=companion` seeds the constructor from a `<Type>Defaults()` function
// the generated package supplies. It is explicit rather than detected because
// the generator cannot see its own output package: routing is not resolved
// until Layout, so probing for the function would mean computing an output path
// the framework owns.
//
// # Per-field declarations
//
// A field seeds its own default with `//testkit:default <expression>`, owned by
// [go.thesmos.sh/testkit/generator/internal/defaults] so a later generator can
// read the same stamp. A field opts out of a setter entirely with a
// `builder:"-"` struct tag, for one a test should never set but which cannot be
// unexported.
//
// # Setter shape follows the field's type
//
// A slice owes a variadic replacing setter and an appending one; a byte slice
// owes a string-accepting setter so a caller need not convert; a map owes a
// replacing setter, a single-entry setter, and a merging one. Everything else
// takes one replacing setter that keeps the field's declared type — a setter
// for `Weekday int` takes `Weekday`, or the declaration was pointless.
//
// # Hazards
//
// Clone copies the slice, byte-slice and map fields so configuring one clone is
// not visible through another. Values held inside those are shared, as are
// pointer fields — which is what a pointer means, and what stops a
// self-referential struct sending the copy into a loop it cannot leave.
package builder
