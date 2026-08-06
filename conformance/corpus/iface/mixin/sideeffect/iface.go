// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package sideeffect is the mixin-axis fixture for the sideeffect mixin, which
// declares that the method mutates state beyond its return value.
//
// The interface carries whatever methods the law needs to be stateable.
// A mixin whose law spans two calls cannot be hosted by a single method:
// there would be nothing to compare against, and the generated subtest
// would pass by having nothing to check.
//
// There is no negated form here. eidos declares the mixin directive
// DenyNegation, because a mixin is opt-in and deleting the directive is
// the suppression (docs/adr/0016).
package sideeffect

import (
	"context"
)

// Mixed is the fixture interface.
//
//testkit:out sideeffecttest/ pkg=sideeffecttest
//testkit:stub
type Mixed interface {
	// Touch returns nothing useful, so its entire effect is out of band. That
	// is what the mixin declares, and Observed is what makes it visible.
	//testkit:mixin sideeffect
	Touch(ctx context.Context, key string) error

	// Observed exposes the out-of-band effect.
	Observed(ctx context.Context, key string) (int, error)
}
