// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package total is the mixin-axis fixture for the total mixin, which
// declares that a callable answers for every input in a named domain rather
// than failing on some of them.
//
// One method is enough here: the claim is about what a single call returns
// for a hostile or unusual input, not about how two calls relate.
//
// There is no negated form here. eidos declares the mixin directive
// DenyNegation, because a mixin is opt-in and deleting the directive is
// the suppression (docs/adr/0016).
package total

import (
	"context"
)

// Mixed is the fixture interface.
//
//testkit:out totaltest/ pkg=totaltest
//testkit:stub
//testkit:suite
//testkit:model
type Mixed interface {
	// Classify answers for any input in the declared domain.
	//testkit:mixin total domain=strings
	Classify(ctx context.Context, in string) (string, error)
}
