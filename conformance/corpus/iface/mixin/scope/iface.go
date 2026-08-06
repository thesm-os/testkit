// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package scope is the mixin-axis fixture for the scope mixin, which
// declares that the effect is confined to a named scope.
//
// The interface carries whatever methods the law needs to be stateable.
// A mixin whose law spans two calls cannot be hosted by a single method:
// there would be nothing to compare against, and the generated subtest
// would pass by having nothing to check.
//
// There is no negated form here. eidos declares the mixin directive
// DenyNegation, because a mixin is opt-in and deleting the directive is
// the suppression (docs/adr/0016).
package scope

import (
	"context"
)

// Mixed is the fixture interface.
//
//testkit:out scopetest/ pkg=scopetest
//testkit:stub
type Mixed interface {
	// Set is confined to the named scope. The law reads back from another
	// scope to prove containment, so a reader is required.
	//testkit:mixin scope name=tenant
	Set(ctx context.Context, scope, key, value string) error

	// Get reads within a scope, which is what makes confinement observable.
	Get(ctx context.Context, scope, key string) (string, error)
}
