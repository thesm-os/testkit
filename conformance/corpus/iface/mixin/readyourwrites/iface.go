// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package readyourwrites is the mixin-axis fixture for the readyourwrites mixin, which
// declares that a client always observes its own writes.
//
// The interface carries a write and a read because the claim relates them:
// a read alone has nothing to be consistent with, and the generated subtest
// would pass by having nothing to check.
//
// There is no negated form here. eidos declares the mixin directive
// DenyNegation, because a mixin is opt-in and deleting the directive is
// the suppression (docs/adr/0016).
package readyourwrites

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
//testkit:out readyourwritestest/ pkg=readyourwritestest
//testkit:stub
//testkit:suite
//testkit:model
type Mixed interface {
	// Store is the write the read is judged against.
	// Store answers the stored state — the Rev it assigned rides back to
	// the caller, which is what lets a trace order this client's writes.
	Store(ctx context.Context, v Value) (Value, error)

	// Get reads a key, never returning a version older than this client wrote.
	//testkit:mixin readyourwrites version=Rev
	Get(ctx context.Context, key string) (Value, error)
}
