// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package eventually is the mixin-axis fixture for the eventually mixin, which
// declares that the effect becomes visible after a quiet period, not immediately.
//
// Both methods have an identical signature and only one carries the
// directive. Holding the shape constant makes the directive the only
// variable, so any difference in generated output is attributable to it and
// to nothing else — and Plain proves the mixin is opt-in rather than inferred
// from the signature.
//
// There is no negated form here: eidos declares the mixin directive
// DenyNegation, because a mixin is opt-in and there is nothing to suppress.
package eventually

import (
	"context"
	"errors"
)

// ErrNotFound is the miss sentinel both methods report.
var ErrNotFound = errors.New("eventually: not found")

// Value is the payload the fixture reads.
type Value struct{ Key, Body string }

// Mixed is the fixture interface.
type Mixed interface {
	//testkit:mixin eventually
	Declared(ctx context.Context, key string) (Value, error)

	// Plain carries no directive, so it must stamp nothing.
	Plain(ctx context.Context, key string) (Value, error)
}
