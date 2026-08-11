// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package concurrent is the mixin-axis fixture for the concurrent mixin, which
// declares that the method is safe to call from several goroutines at once.
//
// The interface carries whatever methods the law needs to be stateable.
// A mixin whose law spans two calls cannot be hosted by a single method:
// there would be nothing to compare against, and the generated subtest
// would pass by having nothing to check.
//
// There is no negated form here. eidos declares the mixin directive
// DenyNegation, because a mixin is opt-in and deleting the directive is
// the suppression (docs/adr/0016).
package concurrent

import (
	"context"
)

// Mixed is the fixture interface.
//
//testkit:out concurrenttest/ pkg=concurrenttest
//testkit:stub
//testkit:suite
type Mixed interface {
	// Bump is driven concurrently by the generated stress. The law is the
	// absence of a race, so the method must mutate something.
	//testkit:mixin concurrent
	Bump(ctx context.Context, key string) error

	// Count observes the mutations, since a race with no observable effect
	// cannot be asserted on.
	Count(ctx context.Context) (int, error)
}
