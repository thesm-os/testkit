// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package writesfollowreads is the mixin-axis fixture for the writesfollowreads mixin, which
// declares that a write made after a read is ordered after the write
// that read observed.
//
// The interface carries a write and a read because the claim relates them:
// a read alone has nothing to be consistent with, and the generated subtest
// would pass by having nothing to check.
//
// There is no negated form here. eidos declares the mixin directive
// DenyNegation, because a mixin is opt-in and deleting the directive is
// the suppression (docs/adr/0016).
package writesfollowreads

import (
	"context"
)

// Value is the payload the store holds.
type Value struct{ Key, Body string }

// Mixed is the fixture interface.
//
//testkit:out writesfollowreadstest/ pkg=writesfollowreadstest
//testkit:stub
//testkit:suite
//testkit:model
type Mixed interface {
	// Store is the write the read is judged against.
	Store(ctx context.Context, v Value) error

	// Get reads the value a subsequent write is ordered against.
	//testkit:mixin writesfollowreads
	Get(ctx context.Context, key string) (Value, error)
}
