// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package noduplicates is the mixin-axis fixture for the noduplicates mixin, which
// declares that a drain yields each element at most once.
//
// The interface carries an append beside the drain: a claim about what a
// drain yields needs something to have gone in, or the generated subtest
// passes over an empty collection.
//
// There is no negated form here. eidos declares the mixin directive
// DenyNegation, because a mixin is opt-in and deleting the directive is
// the suppression (docs/adr/0016).
package noduplicates

import (
	"context"
)

// Value is the element the collection holds.
type Value struct{ Key, Body string }

// Mixed is the fixture interface.
//
//testkit:out noduplicatestest/ pkg=noduplicatestest
//testkit:stub
//testkit:suite
type Mixed interface {
	// Add puts an element in for the drain to yield.
	Add(ctx context.Context, v Value) error

	// Items drains the collection. No element appears twice in one drain.
	//testkit:mixin noduplicates
	Items(ctx context.Context) ([]Value, error)
}
