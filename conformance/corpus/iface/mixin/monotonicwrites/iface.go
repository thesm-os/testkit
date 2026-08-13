// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package monotonicwrites is the mixin-axis fixture for the monotonicwrites mixin, which
// declares that one client's writes are applied in the order it issued them.
//
// The interface carries a write and a read because the claim relates them:
// a read alone has nothing to be consistent with, and the generated subtest
// would pass by having nothing to check.
//
// There is no negated form here. eidos declares the mixin directive
// DenyNegation, because a mixin is opt-in and deleting the directive is
// the suppression (docs/adr/0016).
package monotonicwrites

import (
	"context"
)

// Value is the payload the store holds. Rev is the ordering stamp the
// subject assigns on write — named by the mixin, read by the law.
type Value struct {
	Key, Body string
	Rev       int64
}

// Mixed is the fixture interface.
//
//testkit:out monotonicwritestest/ pkg=monotonicwritestest
//testkit:stub
//testkit:suite
//testkit:model
type Mixed interface {
	// Store is the write the read is judged against.
	Store(ctx context.Context, v Value) error

	// Get reads back what the ordered writes produced.
	//testkit:mixin monotonicwrites version=Rev
	Get(ctx context.Context, key string) (Value, error)
}
