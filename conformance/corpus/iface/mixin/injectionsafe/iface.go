// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package injectionsafe is the mixin-axis fixture for the injectionsafe mixin, which
// declares that a value carrying a control sequence is stored and returned
// as data rather than interpreted.
//
// One method is enough here: the claim is about what a single call returns
// for a hostile or unusual input, not about how two calls relate.
//
// There is no negated form here. eidos declares the mixin directive
// DenyNegation, because a mixin is opt-in and deleting the directive is
// the suppression (docs/adr/0016).
package injectionsafe

import (
	"context"
)

// Mixed is the fixture interface.
//
//testkit:out injectionsafetest/ pkg=injectionsafetest
//testkit:stub
//testkit:suite
//testkit:model
type Mixed interface {
	// Store stores a value and hands back what was stored.
	//testkit:mixin injectionsafe
	Store(ctx context.Context, in string) (string, error)
}
