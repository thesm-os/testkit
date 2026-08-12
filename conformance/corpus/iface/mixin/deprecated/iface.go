// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package deprecated is the mixin-axis fixture for the deprecated mixin, which
// declares that the method is scheduled for removal and callers should migrate.
//
// The interface carries whatever methods the law needs to be stateable.
// A mixin whose law spans two calls cannot be hosted by a single method:
// there would be nothing to compare against, and the generated subtest
// would pass by having nothing to check.
//
// There is no negated form here. eidos declares the mixin directive
// DenyNegation, because a mixin is opt-in and deleting the directive is
// the suppression (docs/adr/0016).
package deprecated

import (
	"context"
)

// Mixed is the fixture interface.
//
//testkit:out deprecatedtest/ pkg=deprecatedtest
//testkit:stub
//testkit:suite
//testkit:model
type Mixed interface {
	// Old carries no runtime law: the mixin is an annotation the generator
	// surfaces, so the signature is deliberately unremarkable.
	//testkit:mixin deprecated
	Old(ctx context.Context, key string) (string, error)

	// New is the replacement the annotation points callers towards.
	New(ctx context.Context, key string) (string, error)
}
