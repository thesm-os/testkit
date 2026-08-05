// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package errors is the mixin-axis fixture for the errors mixin, which
// declares that the method reports misses through a declared sentinel.
//
// The interface carries whatever methods the law needs to be stateable.
// A mixin whose law spans two calls cannot be hosted by a single method:
// there would be nothing to compare against, and the generated subtest
// would pass by having nothing to check.
//
// There is no negated form here. eidos declares the mixin directive
// DenyNegation, because a mixin is opt-in and deleting the directive is
// the suppression (docs/adr/0016).
package errors

import (
	"context"
	"errors"
)

// ErrNotFound is the sentinel the mixin declares. A fixture without one leaves
// the generated assertion nothing to compare against.
var ErrNotFound = errors.New("errors: not found")

// Mixed is the fixture interface.
type Mixed interface {
	// Get reports [ErrNotFound] and nothing else for a miss. The sentinel has
	// to be declared in the package for the assertion to reference it.
	//testkit:mixin errors
	Get(ctx context.Context, key string) (string, error)
}
