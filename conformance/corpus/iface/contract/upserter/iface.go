// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package upserter is the contract-axis fixture for the upserter contract:
// an upsert paired with the read that observes it.
//
// Every role the contract declares is present, because a contract is a
// multi-callable protocol and a fixture missing a partner exercises the
// validator's failure path rather than the contract itself.
package upserter

import (
	"context"
)

// Value is the payload the contract's roles carry.
type Value struct{ Key, Body string }

// Contract is the fixture interface.
//
//testkit:out upsertertest/ pkg=upsertertest
//testkit:stub
//testkit:suite
//testkit:model
type Contract interface {
	// Put is the upserter contract's writer role, and hosts the directive
	// that names its partners.
	//testkit:contract upserter role=writer reader=Get
	Put(ctx context.Context, v Value) error

	// Get is the upserter contract's reader role.
	Get(ctx context.Context, key string) (Value, error)
}
