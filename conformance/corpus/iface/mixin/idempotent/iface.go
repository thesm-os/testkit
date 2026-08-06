// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package idempotent is the mixin-axis fixture for the idempotent mixin, which
// declares that repeating the call leaves the observable state unchanged.
//
// The interface carries whatever methods the law needs to be stateable.
// A mixin whose law spans two calls cannot be hosted by a single method:
// there would be nothing to compare against, and the generated subtest
// would pass by having nothing to check.
//
// There is no negated form here. eidos declares the mixin directive
// DenyNegation, because a mixin is opt-in and deleting the directive is
// the suppression (docs/adr/0016).
package idempotent

import (
	"context"
)

// Mixed is the fixture interface.
//
//testkit:out idempotenttest/ pkg=idempotenttest
//testkit:stub
type Mixed interface {
	// Put must be safe to repeat. The law compares state after one call with
	// state after two, so a reader is required to make that comparison.
	//testkit:mixin idempotent
	Put(ctx context.Context, key, value string) error

	// Read observes the state the law compares.
	Read(ctx context.Context, key string) (string, error)
}
