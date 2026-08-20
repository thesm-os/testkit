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

// A fixture without a declared sentinel leaves the generated assertion nothing
// to compare against.
var (
	// ErrNotFound is the sentinel the mixin declares.
	ErrNotFound = errors.New("errors: not found")

	// ErrGone is a second sentinel, so the fault directive's variadic form is
	// exercised by a fixture rather than only by a unit test.
	ErrGone = errors.New("errors: gone")
)

// Mixed is the fixture interface.
//
//testkit:out errorstest/ pkg=errorstest
//testkit:stub
//testkit:suite
//testkit:model
type Mixed interface {
	// Get reports [ErrNotFound] and nothing else for a miss. The sentinel has
	// to be declared in the package for the assertion to reference it.
	//
	// The notfound mixin is what makes the miss derivable: nothing on this
	// interface writes, so without a declared sentinel no draw would be one
	// nothing supplied and the rule refuses. The fault directive is a
	// different key — it says what the DOUBLE can be told to return.
	//testkit:mixin errors
	//testkit:mixin notfound sentinel=ErrNotFound
	//testkit:fault ErrNotFound ErrGone
	Get(ctx context.Context, key string) (string, error)
}
