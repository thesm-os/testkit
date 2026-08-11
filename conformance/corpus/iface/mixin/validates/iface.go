// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package validates is the mixin-axis fixture for the validates mixin, which
// declares that invalid input is rejected rather than stored.
//
// The interface carries whatever methods the law needs to be stateable.
// A mixin whose law spans two calls cannot be hosted by a single method:
// there would be nothing to compare against, and the generated subtest
// would pass by having nothing to check.
//
// There is no negated form here. eidos declares the mixin directive
// DenyNegation, because a mixin is opt-in and deleting the directive is
// the suppression (docs/adr/0016).
package validates

import (
	"context"
)

// Payload is what Validate accepts or refuses.
type Payload struct{ Key, Body string }

// Mixed is the fixture interface.
//
//testkit:out validatestest/ pkg=validatestest
//testkit:stub
//testkit:suite
//testkit:model
type Mixed interface {
	// Store rejects what Validate refuses. The fn parameter resolves to a
	// sibling, so Validate has to exist here for the reference to bind.
	//testkit:mixin validates fn=Validate
	Store(ctx context.Context, v Payload) error

	// Validate is the predicate fn names.
	Validate(v Payload) error

	// Read proves a rejected value was not stored.
	Read(ctx context.Context, key string) (Payload, error)
}
