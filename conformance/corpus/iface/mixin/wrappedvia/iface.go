// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package wrappedvia is the mixin-axis fixture for the wrappedvia mixin, which
// declares that the returned error wraps a declared cause.
//
// The interface carries whatever methods the law needs to be stateable.
// A mixin whose law spans two calls cannot be hosted by a single method:
// there would be nothing to compare against, and the generated subtest
// would pass by having nothing to check.
//
// There is no negated form here. eidos declares the mixin directive
// DenyNegation, because a mixin is opt-in and deleting the directive is
// the suppression (docs/adr/0016).
package wrappedvia

import (
	"context"
)

// Mixed is the fixture interface.
//
//testkit:out wrappedviatest/ pkg=wrappedviatest
//testkit:stub
//testkit:suite
type Mixed interface {
	// Open wraps whatever Cause returns. The fn parameter resolves to a
	// sibling, so the cause has to be reachable from this interface.
	//testkit:mixin wrappedvia fn=Cause
	Open(ctx context.Context, name string) error

	// Cause is the wrapped error fn names.
	Cause(ctx context.Context) error
}
