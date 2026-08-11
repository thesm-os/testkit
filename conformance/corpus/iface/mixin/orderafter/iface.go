// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package orderafter is the mixin-axis fixture for the orderafter mixin, which
// declares that the call is only valid after a named predecessor has run.
//
// The interface carries whatever methods the law needs to be stateable.
// A mixin whose law spans two calls cannot be hosted by a single method:
// there would be nothing to compare against, and the generated subtest
// would pass by having nothing to check.
//
// There is no negated form here. eidos declares the mixin directive
// DenyNegation, because a mixin is opt-in and deleting the directive is
// the suppression (docs/adr/0016).
package orderafter

import (
	"context"
)

// Mixed is the fixture interface.
//
//testkit:out orderaftertest/ pkg=orderaftertest
//testkit:stub
//testkit:suite
type Mixed interface {
	// Commit is valid only after Prepare. The fn parameter names the
	// predecessor, and the resolver resolves it to a sibling — so Prepare has
	// to exist in this interface or the reference dangles.
	//testkit:mixin orderafter fn=Prepare
	Commit(ctx context.Context) error

	// Prepare is the predecessor fn names.
	Prepare(ctx context.Context) error
}
