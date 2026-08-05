// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package monotonic is the mixin-axis fixture for the monotonic mixin, which
// declares that successive reads never go backwards.
//
// The interface carries whatever methods the law needs to be stateable.
// A mixin whose law spans two calls cannot be hosted by a single method:
// there would be nothing to compare against, and the generated subtest
// would pass by having nothing to check.
//
// There is no negated form here. eidos declares the mixin directive
// DenyNegation, because a mixin is opt-in and deleting the directive is
// the suppression (docs/adr/0016).
package monotonic

import (
	"context"
)

// Mixed is the fixture interface.
type Mixed interface {
	// Version must not decrease across calls. A single read cannot violate
	// that, so the law is over a sequence and Advance is what moves it.
	//testkit:mixin monotonic
	Version(ctx context.Context) (int64, error)

	// Advance moves the value the law watches.
	Advance(ctx context.Context) error
}
