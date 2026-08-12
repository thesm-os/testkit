// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package leakfree is the mixin-axis fixture for the leakfree mixin, which
// declares that a resource acquired is a resource released — repeating the
// cycle leaves nothing outstanding.
//
// The interface carries both halves of the cycle and a way to observe the
// balance, because the claim is about the pair: an acquire alone leaks by
// definition, and a release alone has nothing to release.
//
// There is no negated form here. eidos declares the mixin directive
// DenyNegation, because a mixin is opt-in and deleting the directive is
// the suppression (docs/adr/0016).
package leakfree

import (
	"context"
)

// Mixed is the fixture interface.
//
//testkit:out leakfreetest/ pkg=leakfreetest
//testkit:stub
//testkit:suite
//testkit:model
type Mixed interface {
	// Acquire takes the resource. The mixin names both halves of the cycle
	// whose balance is the claim.
	//testkit:mixin leakfree open=Acquire close=Release
	Acquire(ctx context.Context) error

	// Release gives it back.
	Release(ctx context.Context) error

	// Outstanding observes the balance, which is what makes a leak visible
	// rather than merely eventual.
	Outstanding(ctx context.Context) (int, error)
}
