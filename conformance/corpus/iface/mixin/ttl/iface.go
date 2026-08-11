// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package ttl is the mixin-axis fixture for the ttl mixin, which declares that
// stored data stops being readable once its lifetime elapses.
//
// The directive names everything the claim needs: the lifetime, the pair of
// callables it governs, and the sentinel a lapsed read reports. It is the one
// classification using all three reference kinds at once — a verbatim value, a
// pair of sibling callables, and a package-level var.
//
// There is no negated form here. eidos declares the mixin directive
// DenyNegation, because a mixin is opt-in and deleting the directive is
// the suppression (docs/adr/0016).
package ttl

import (
	"context"
	"errors"
)

// ErrExpired is what a read past the lifetime reports.
//
// Package-level because the directive names it: a sentinel is a var rather
// than a callable, which is why it resolves through its own scope.
var ErrExpired = errors.New("ttl: entry expired")

// Value is the payload the store holds.
type Value struct{ Key, Body string }

// Mixed is the fixture interface.
//
//testkit:out ttltest/ pkg=ttltest
//testkit:stub
//testkit:suite
//testkit:model
type Mixed interface {
	// Put stores a value, starting its lifetime.
	Put(ctx context.Context, v Value) error

	// Read returns it while the lifetime holds, and the sentinel after.
	//testkit:mixin ttl duration=1m put=Put read=Read notfound=ErrExpired
	Read(ctx context.Context, key string) (Value, error)
}
