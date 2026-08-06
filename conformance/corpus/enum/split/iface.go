// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package split declares its enum's type and its values in different files.
//
// A generator that read only the file carrying the type would find an enum
// with no variants, and one that read only the constant block would find a
// block with no type. Neither failure is visible in a single-file fixture,
// because there the two are always found together.
//
// This is an ordinary way to write a large enum — the type and its docs in one
// place, a long generated or grouped constant block in another — so the
// arrangement has to work rather than merely not crash.
package split

// Signal is the enumerated type, declared alone.
//
//testkit:enum
type Signal int
