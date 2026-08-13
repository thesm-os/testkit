// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package indexed is the mixin-axis fixture for the indexed mixin, which
// declares that an integer parameter addresses a position in the collection
// another method sizes.
//
// The shape the fixture axis had none of, and the one that made the gap
// visible: a derived value for a bare `int` is a number invented from the
// type, and an index invented from the type is out of range for every
// collection smaller than it. Nothing in the signature says the parameter is
// a position rather than a magnitude, so nothing could have known.
//
// `by=Len` names the sizing method as a sibling callable, so a misspelling
// fails at the directive. It cannot supply the bound: the size is a fact
// about the seeded subject at run time, not about the declaration.
//
// There is no negated form here. eidos declares the mixin directive
// DenyNegation, because a mixin is opt-in and deleting the directive is
// the suppression (docs/adr/0016).
package indexed

import (
	"context"
)

// Value is the element the positions address.
type Value struct{ Key, Body string }

// Ranked is the fixture interface.
//
//testkit:out indexedtest/ pkg=indexedtest
//testkit:stub
//testkit:suite
//testkit:model
type Ranked interface {
	// Add appends an element, so the positions have something to address.
	Add(ctx context.Context, v Value) error

	// Len reports how many elements the positions address — the bound the
	// mixin names, and an ordinary read the sequences still drive.
	Len(ctx context.Context) (int, error)

	// At answers the element at a position, and reports a miss for one the
	// collection does not hold.
	//testkit:mixin indexed by=Len
	At(ctx context.Context, i int) (Value, error)
}
