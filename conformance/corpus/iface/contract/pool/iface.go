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

// Stats is the accounting the pool laws balance — on the interface per
// the database/sql.DBStats precedent, and named by the contract
// directive so the generator knows where the numbers live.
type Stats struct{ Gets, Puts, Outstanding int }

// Contract is the fixture interface.
//
//testkit:out pooltest/ pkg=pooltest
//testkit:stub
//testkit:suite
//testkit:model
type Contract interface {
	// Get is the pool contract's get role, and hosts the directive
	// that names its partners — the put beside it, and the optional
	// accounting observation the balanced laws read.
	//testkit:contract pool role=get put=Put stats=Stats
	Get(ctx context.Context) (Value, error)

	// Put is the pool contract's put role.
	Put(ctx context.Context, v Value) error

	// Stats is the pool contract's stats role: the accounting the
	// balanced and leak-free laws verify against the cycle count.
	Stats(ctx context.Context) (Stats, error)
}
