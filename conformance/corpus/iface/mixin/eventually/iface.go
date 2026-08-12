// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package eventually is the mixin-axis fixture for the eventually mixin, which
// declares that the effect becomes visible after a quiet period rather than immediately.
//
// The interface carries whatever methods the law needs to be stateable.
// A mixin whose law spans two calls cannot be hosted by a single method:
// there would be nothing to compare against, and the generated subtest
// would pass by having nothing to check.
//
// There is no negated form here. eidos declares the mixin directive
// DenyNegation, because a mixin is opt-in and deleting the directive is
// the suppression (docs/adr/0016).
package eventually

import (
	"context"
)

// Mixed is the fixture interface.
//
//testkit:out eventuallytest/ pkg=eventuallytest
//testkit:stub
//testkit:suite
//testkit:model
type Mixed interface {
	// Publish is not observable on return. The law is that it becomes so,
	// which needs a settle step and a read to observe after it.
	//testkit:mixin eventually
	Publish(ctx context.Context, item string) error

	// Settle advances the quiet window the law waits out.
	Settle(ctx context.Context) error

	// Items observes the eventual state.
	Items(ctx context.Context) ([]string, error)
}
