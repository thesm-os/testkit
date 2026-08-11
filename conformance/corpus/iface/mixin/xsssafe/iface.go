// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package xsssafe is the mixin-axis fixture for the xsssafe mixin, which
// declares that a value rendered into markup cannot close a tag or open a
// script.
//
// One method is enough here: the claim is about what a single call returns
// for a hostile or unusual input, not about how two calls relate.
//
// There is no negated form here. eidos declares the mixin directive
// DenyNegation, because a mixin is opt-in and deleting the directive is
// the suppression (docs/adr/0016).
package xsssafe

import (
	"context"
)

// Mixed is the fixture interface.
//
//testkit:out xsssafetest/ pkg=xsssafetest
//testkit:stub
//testkit:suite
type Mixed interface {
	// Render escapes a raw value for embedding in markup.
	//testkit:mixin xsssafe
	Render(ctx context.Context, in string) (string, error)
}
