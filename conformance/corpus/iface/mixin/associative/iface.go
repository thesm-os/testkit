// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package associative is the mixin-axis fixture for the associative mixin, which
// declares that the order of folding is irrelevant when the grouping
// changes.
//
// The interface carries a total beside the step: an algebraic claim about a
// fold needs somewhere to read the fold's result, or the generated subtest
// passes over a state nothing can see.
//
// There is no negated form here. eidos declares the mixin directive
// DenyNegation, because a mixin is opt-in and deleting the directive is
// the suppression (docs/adr/0016).
package associative

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
//testkit:out associativetest/ pkg=associativetest
//testkit:stub
//testkit:suite
//testkit:model
type Mixed interface {
	// Apply folds one delta in. Regrouping the folds does not change the total.
	//testkit:mixin associative
	Apply(ctx context.Context, d Delta) error

	// Total observes the fold, which is what makes the claim about Apply
	// checkable at all.
	Total(ctx context.Context) (int, error)
}
