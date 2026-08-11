// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package pool is the contract-axis fixture for the pool contract:
// a pool whose every Get is returned by a Put.
//
// Every role the contract declares is present, because a contract is a
// multi-callable protocol and a fixture missing a partner exercises the
// validator's failure path rather than the contract itself.
package pool

import (
	"context"
)

// Value is the payload the contract's roles carry.
type Value struct{ Key, Body string }

// Contract is the fixture interface.
//
//testkit:out pooltest/ pkg=pooltest
//testkit:stub
//testkit:suite
type Contract interface {
	// Get is the pool contract's get role, and hosts the directive
	// that names its partners.
	//testkit:contract pool role=get put=Put
	Get(ctx context.Context) (Value, error)

	// Put is the pool contract's put role.
	Put(ctx context.Context, v Value) error
}
