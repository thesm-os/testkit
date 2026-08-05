// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package lifecycleafterclose is the mixin-axis fixture for the lifecycleafterclose mixin, which
// declares that calls after teardown report the closed sentinel.
//
// Both methods have an identical signature and only one carries the
// directive. Holding the shape constant makes the directive the only
// variable, so any difference in generated output is attributable to it and
// to nothing else — and Plain proves the mixin is opt-in rather than inferred
// from the signature.
//
// There is no negated form here: eidos declares the mixin directive
// DenyNegation, because a mixin is opt-in and there is nothing to suppress.
package lifecycleafterclose

import (
	"context"
	"errors"
)

// ErrNotFound is the miss sentinel both methods report.
var ErrNotFound = errors.New("lifecycleafterclose: not found")

// Value is the payload the fixture reads.
type Value struct{ Key, Body string }

// Mixed is the fixture interface.
type Mixed interface {
	//testkit:mixin lifecycleafterclose
	Declared(ctx context.Context, key string) (Value, error)

	// Plain carries no directive, so it must stamp nothing.
	Plain(ctx context.Context, key string) (Value, error)
}
