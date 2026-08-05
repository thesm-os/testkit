// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package bare is the minimal half of the errors-kind corpus: package-level
// sentinels and nothing else.
//
// The generator emits an umbrella test covering prefix, uniqueness,
// non-overlap and unwrap-chain for any sentinel set, then appends further
// subtests for each optional method it detects. With no custom error types
// here there is no Is to check for self-consistency, no Unwrap to traverse,
// and no format string to round-trip — so this fixture is the floor, and
// [go.thesmos.sh/testkit/conformance/corpus/errors/store] is the ceiling.
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
