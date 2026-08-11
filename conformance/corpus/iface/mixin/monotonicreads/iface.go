// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package monotonicreads is the mixin-axis fixture for the monotonicreads mixin, which
// declares that successive reads by one client never move backwards.
//
// The interface carries a write and a read because the claim relates them:
// a read alone has nothing to be consistent with, and the generated subtest
// would pass by having nothing to check.
//
// There is no negated form here. eidos declares the mixin directive
// DenyNegation, because a mixin is opt-in and deleting the directive is
// the suppression (docs/adr/0016).
package monotonicreads

import (
	"context"
)

// Value is the payload the store holds.
type Value struct{ Key, Body string }

// Mixed is the fixture interface.
//
//testkit:out monotonicreadstest/ pkg=monotonicreadstest
//testkit:stub
//testkit:suite
type Mixed interface {
	// Store is the write the read is judged against.
	Store(ctx context.Context, v Value) error

	// Get reads a key. Once a client has seen a version, it never sees an older one.
	//testkit:mixin monotonicreads
	Get(ctx context.Context, key string) (Value, error)
}
