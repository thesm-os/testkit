// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package cache is the contract-axis fixture for the cache contract:
// a cache in front of a backing store.
//
// Every role the contract declares is present, because a contract is a
// multi-callable protocol and a fixture missing a partner exercises the
// validator's failure path rather than the contract itself.
package cache

import (
	"context"
)

// Value is the payload the contract's roles carry.
type Value struct{ Key, Body string }

// Contract is the fixture interface.
type Contract interface {
	// Lookup is the cache contract's cache role, and hosts the directive
	// that names its partners.
	//testkit:contract cache role=cache backing=Fetch
	Lookup(ctx context.Context, key string) (Value, error)

	// Fetch is the cache contract's backing role.
	Fetch(ctx context.Context, key string) (Value, error)
}
