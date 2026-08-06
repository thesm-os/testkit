// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package bare is the minimal half of the errors-kind corpus: package-level
// sentinels and nothing else.
//
// The generator emits an umbrella test covering, for any sentinel set: that
// every value is assigned, that every message is non-empty and carries the
// package prefix, that no message repeats or begins with the whole of
// another, and that no sentinel satisfies errors.Is against a sibling. It
// then appends further subtests for each optional method it detects. With no
// custom error types here there is no Is to check for self-consistency, no
// Unwrap to traverse, and no format string to round-trip — so this fixture is
// the floor, and [go.thesmos.sh/testkit/conformance/corpus/errors/store] is
// the ceiling.
//
// No survives-wrapping check is emitted for any of them. errors.Is compares
// identity before it consults anything a type declares, so every sentinel
// passes %w and errors.Join unconditionally — including one whose Is always
// refuses. Such a check asserts the behaviour of the standard library rather
// than of this package, which is why it is absent rather than merely unused.
//
// A corpus holding only the rich case cannot tell "the optional subtests were
// correctly omitted" from "the optional subtests were silently dropped".
//
//testkit:sentinel
package bare

import "errors"

// The sentinel set. Three is the minimum that makes uniqueness and
// non-overlap meaningful: with two, a pair that accidentally aliases is
// indistinguishable from a pair that overlaps.
var (
	// ErrEmpty reports an operation on an empty collection.
	ErrEmpty = errors.New("bare: empty")

	// ErrFull reports an operation on a saturated collection.
	ErrFull = errors.New("bare: full")

	// ErrInvalid reports input the package rejects.
	ErrInvalid = errors.New("bare: invalid")
)
