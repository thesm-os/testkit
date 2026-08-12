// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package sample is the mixin-axis fixture for the sample mixin, which
// declares that a declared value is used as generated input rather than a random draw.
//
// The interface carries whatever methods the law needs to be stateable.
// A mixin whose law spans two calls cannot be hosted by a single method:
// there would be nothing to compare against, and the generated subtest
// would pass by having nothing to check.
//
// There is no negated form here. eidos declares the mixin directive
// DenyNegation, because a mixin is opt-in and deleting the directive is
// the suppression (docs/adr/0016).
package sample

import (
	"context"
)

// Mixed is the fixture interface.
//
//testkit:out sampletest/ pkg=sampletest
//testkit:stub
//testkit:suite
//testkit:model
type Mixed interface {
	// Process takes an input the mixin pins. The parameter is the point: with
	// no argument there is nothing for a sample to replace.
	//testkit:mixin sample builder=NewInput
	Process(ctx context.Context, input string) (string, error)

	// NewInput is the builder the mixin names. It sits on the interface
	// because that is where the sibling resolver looks: a mixin on a method
	// resolves its partner against the owner's method list, so a package-level
	// factory would not be found however well it fitted.
	NewInput(ctx context.Context) (string, error)
}
