// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package atomic is the mixin-axis fixture for the atomic mixin, which
// declares that a failed write leaves no partial state, so a reader never sees half of it.
//
// The interface carries whatever methods the law needs to be stateable.
// A mixin whose law spans two calls cannot be hosted by a single method:
// there would be nothing to compare against, and the generated subtest
// would pass by having nothing to check.
//
// There is no negated form here. eidos declares the mixin directive
// DenyNegation, because a mixin is opt-in and deleting the directive is
// the suppression (docs/adr/0016).
package atomic

import (
	"context"
)

// Entry is the two-field write the mixin holds together: a failed write
// applies neither field, and Read answers both so a half-applied one is
// observable.
type Entry struct{ Key, Left, Right string }

// Mixed is the fixture interface.
//
//testkit:out atomictest/ pkg=atomictest
//testkit:stub
//testkit:suite
//testkit:model
type Mixed interface {
	// Write applies both of the entry's fields or neither. One input
	// rather than several, because the law's claim is about one write's
	// halves — the fields travel together so the observation can too.
	//testkit:mixin atomic
	Write(ctx context.Context, e Entry) error

	// Read returns the whole entry, so a partial write is observable.
	Read(ctx context.Context, key string) (Entry, error)
}
