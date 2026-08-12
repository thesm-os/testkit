// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package hooks is the mixin-axis fixture for the hooks mixin, which
// declares that the method fires registered callbacks as part of its work.
//
// The interface carries whatever methods the law needs to be stateable.
// A mixin whose law spans two calls cannot be hosted by a single method:
// there would be nothing to compare against, and the generated subtest
// would pass by having nothing to check.
//
// There is no negated form here. eidos declares the mixin directive
// DenyNegation, because a mixin is opt-in and deleting the directive is
// the suppression (docs/adr/0016).
package hooks

import (
	"context"
)

// Mixed is the fixture interface.
//
//testkit:out hookstest/ pkg=hookstest
//testkit:stub
//testkit:suite
//testkit:model
type Mixed interface {
	// Fire invokes whatever OnEvent registered. Without a registration
	// method there is no way to observe that it did.
	//testkit:mixin hooks register=OnEvent
	Fire(ctx context.Context, event string) error

	// OnEvent registers the callback Fire invokes.
	OnEvent(fn func(event string))
}
