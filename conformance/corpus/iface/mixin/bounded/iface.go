// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package bounded is the mixin-axis fixture for the bounded mixin, which
// declares that the returned collection never exceeds a declared ceiling.
//
// The interface carries whatever methods the law needs to be stateable.
// A mixin whose law spans two calls cannot be hosted by a single method:
// there would be nothing to compare against, and the generated subtest
// would pass by having nothing to check.
//
// There is no negated form here. eidos declares the mixin directive
// DenyNegation, because a mixin is opt-in and deleting the directive is
// the suppression (docs/adr/0016).
package bounded

import (
	"context"
)

// Mixed is the fixture interface.
//
//testkit:out boundedtest/ pkg=boundedtest
//testkit:stub
//testkit:suite
type Mixed interface {
	// List carries the ceiling as a parameter, because the law cannot be
	// stated without a number to compare against.
	//testkit:mixin bounded limit=100
	List(ctx context.Context) ([]string, error)
}
