// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package idempotentclose is the mixin-axis fixture for the idempotent
// mixin on a teardown. The mixin already has a write-shaped fixture; this
// one carries the same claim on a lifecycle method, where "repeating the
// call changes nothing" is the double-close a deferred cleanup relies on.
//
// The interface pairs the teardown with an observation, because the law
// reads state across the second close and a Close-only interface offers
// no method to read it through.
//
// There is no negated form here. eidos declares the mixin directive
// DenyNegation, because a mixin is opt-in and deleting the directive is
// the suppression (docs/adr/0016).
package idempotentclose

import (
	"context"
)

// Closer is the fixture interface.
//
//testkit:out idempotentclosetest/ pkg=idempotentclosetest
//testkit:stub
//testkit:suite
//testkit:model
type Closer interface {
	// Close tears down, and declares the teardown safe to repeat.
	//testkit:mixin idempotent
	Close(ctx context.Context) error

	// Stats reports open resources — the observation the second close
	// must leave unchanged, which a Close-only interface cannot state.
	Stats(ctx context.Context) (int, error)
}
