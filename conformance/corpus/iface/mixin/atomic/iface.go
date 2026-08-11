// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package atomic is the mixin-axis fixture for the atomic mixin, which
// declares that a failed write leaves no partial state, so a reader never sees half of it.
//
// The interface carries whatever methods the law needs to be stateable.
// A mixin whose law spans two calls cannot be hosted by a single method:
// there would be nothing to compare against, and the generated subtest
// would pass by having nothing to check.
//
// There is no negated form here. eidos declares the mixin directive
// DenyNegation, because a mixin is opt-in and deleting the directive is
// the suppression (docs/adr/0016).
package atomic

import (
	"context"
)

// Mixed is the fixture interface.
//
//testkit:out atomictest/ pkg=atomictest
//testkit:stub
//testkit:suite
type Mixed interface {
	// Write applies both fields or neither. Observing one without the other
	// is the violation, which is why Read has to return them together.
	//testkit:mixin atomic
	Write(ctx context.Context, key, left, right string) error

	// Read returns both fields, so a partial write is observable.
	Read(ctx context.Context, key string) (left, right string, err error)
}
