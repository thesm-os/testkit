// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package lease is the contract-axis fixture for the lease contract:
// an acquire balanced by exactly one release.
//
// Every role the contract declares is present, because a contract is a
// multi-callable protocol and a fixture missing a partner exercises the
// validator's failure path rather than the contract itself.
package lease

import (
	"context"
)

// Value is the payload the contract's roles carry.
type Value struct{ Key, Body string }

// Contract is the fixture interface.
//
//testkit:out leasetest/ pkg=leasetest
//testkit:stub
//testkit:suite
//testkit:model
type Contract interface {
	// Acquire is the lease contract's acquire role, and hosts the directive
	// that names its partners.
	//testkit:contract lease role=acquire release=Release
	Acquire(ctx context.Context, key string) error

	// Release is the lease contract's release role.
	Release(ctx context.Context, key string) error
}
