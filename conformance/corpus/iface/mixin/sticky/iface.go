// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package sticky is the mixin-axis fixture for the sticky mixin, which
// declares that once a key resolves, it keeps resolving to the same value.
//
// The interface carries a write beside the read because the claim relates
// them: a read alone has nothing to be judged against.
//
// There is no negated form here. eidos declares the mixin directive
// DenyNegation, because a mixin is opt-in and deleting the directive is
// the suppression (docs/adr/0016).
package sticky

import (
	"context"
)

// Value is the payload the store holds.
type Value struct{ Key, Body string }

// Mixed is the fixture interface.
//
//testkit:out stickytest/ pkg=stickytest
//testkit:stub
//testkit:suite
//testkit:model
type Mixed interface {
	// Store is the write the read is judged against.
	Store(ctx context.Context, v Value) error

	// Get reads a key. The first value it resolves to is the one it keeps.
	//testkit:mixin sticky key=key
	Get(ctx context.Context, key string) (Value, error)
}
