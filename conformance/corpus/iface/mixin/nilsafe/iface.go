// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package nilsafe is the mixin-axis fixture for the nilsafe mixin, which
// declares that a nil argument is handled rather than panicking.
//
// The interface carries whatever methods the law needs to be stateable.
// A mixin whose law spans two calls cannot be hosted by a single method:
// there would be nothing to compare against, and the generated subtest
// would pass by having nothing to check.
//
// There is no negated form here. eidos declares the mixin directive
// DenyNegation, because a mixin is opt-in and deleting the directive is
// the suppression (docs/adr/0016).
package nilsafe

import (
	"context"
)

// Payload is taken by pointer so the nil case exists at all.
type Payload struct{ Key, Body string }

// Mixed is the fixture interface.
type Mixed interface {
	// Store takes a pointer, which is what makes nil expressible. A
	// value parameter would leave the law with no nil to pass.
	//testkit:mixin nilsafe
	Store(ctx context.Context, v *Payload) error
}
