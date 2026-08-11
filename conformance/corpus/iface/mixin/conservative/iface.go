// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package conservative is the mixin-axis fixture for the conservative mixin, which
// declares that the operation preserves a quantity the interface names.
//
// The interface carries a total beside the step: an algebraic claim about a
// fold needs somewhere to read the fold's result, or the generated subtest
// passes over a state nothing can see.
//
// There is no negated form here. eidos declares the mixin directive
// DenyNegation, because a mixin is opt-in and deleting the directive is
// the suppression (docs/adr/0016).
package conservative

import (
	"context"
)

// Delta is one contribution to the fold.
type Delta struct {
	Key    string
	Amount int
}

// Mixed is the fixture interface.
//
//testkit:out conservativetest/ pkg=conservativetest
//testkit:stub
//testkit:suite
type Mixed interface {
	// Apply moves a quantity without creating or destroying any of it.
	//testkit:mixin conservative field=Amount
	Apply(ctx context.Context, d Delta) error

	// Total observes the fold, which is what makes the claim about Apply
	// checkable at all.
	Total(ctx context.Context) (int, error)
}
