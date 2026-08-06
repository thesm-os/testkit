// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package timeout is the mixin-axis fixture for the timeout mixin, which
// declares that the method returns within a declared budget.
//
// The interface carries whatever methods the law needs to be stateable.
// A mixin whose law spans two calls cannot be hosted by a single method:
// there would be nothing to compare against, and the generated subtest
// would pass by having nothing to check.
//
// There is no negated form here. eidos declares the mixin directive
// DenyNegation, because a mixin is opt-in and deleting the directive is
// the suppression (docs/adr/0016).
package timeout

import (
	"context"
)

// Mixed is the fixture interface.
//
//testkit:out timeouttest/ pkg=timeouttest
//testkit:stub
type Mixed interface {
	// Slow carries the budget as a parameter, because "within a budget" is
	// not a statement until a duration is named.
	//testkit:mixin timeout duration=5s
	Slow(ctx context.Context, key string) error
}
