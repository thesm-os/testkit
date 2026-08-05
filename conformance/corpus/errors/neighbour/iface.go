// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package neighbour exists to be the target of another package's
// non-overlap declaration.
//
// The `sentinel-no-overlap-with` directive names a second package whose
// sentinels must not satisfy errors.Is against this one's. That check needs
// two packages by construction, so a corpus with a single errors fixture
// leaves the directive unexercised — it would parse, name nothing reachable,
// and pass.
//
// The sentinels here deliberately read like a store's. Distinct values that
// describe the same conditions are exactly the case where a copy-paste between
// packages produces an accidental alias, which is what the cross-package check
// exists to catch.
//
//testkit:sentinel
package neighbour

import "errors"

var (
	// ErrNotFound reports a missing key. Its message differs from the store
	// package's by prefix alone, which is the near-miss the check must accept
	// as distinct.
	ErrNotFound = errors.New("neighbour: not found")

	// ErrConflict reports a losing write.
	ErrConflict = errors.New("neighbour: conflict")
)
