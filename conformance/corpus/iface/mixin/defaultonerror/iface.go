// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package defaultonerror is the mixin-axis fixture for the defaultonerror mixin, which
// declares that a failed read hands back a stated default rather than a
// partly-populated value.
//
// The interface carries a write beside the read because the claim relates
// them: a read alone has nothing to be judged against.
//
// There is no negated form here. eidos declares the mixin directive
// DenyNegation, because a mixin is opt-in and deleting the directive is
// the suppression (docs/adr/0016).
package defaultonerror

import (
	"context"
)

// Value is the payload the store holds.
type Value struct{ Key, Body string }

// Mixed is the fixture interface.
//
//testkit:out defaultonerrortest/ pkg=defaultonerrortest
//testkit:stub
//testkit:suite
type Mixed interface {
	// Store is the write the read is judged against.
	Store(ctx context.Context, v Value) error

	// Get reads a key, answering an absent one with the default rather than with
	// whatever the failed lookup left behind.
	//testkit:mixin defaultonerror
	Get(ctx context.Context, key string) (Value, error)
}
