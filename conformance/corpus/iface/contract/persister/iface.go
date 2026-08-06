// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package persister is the contract-axis fixture for the persister contract:
// a write paired with the read that observes it.
//
// Every role the contract declares is present, because a contract is a
// multi-callable protocol and a fixture missing a partner exercises the
// validator's failure path rather than the contract itself.
package persister

import (
	"context"
)

// Value is the payload the contract's roles carry.
type Value struct{ Key, Body string }

// Contract is the fixture interface.
//
//testkit:out persistertest/ pkg=persistertest
//testkit:stub
type Contract interface {
	// Put is the persister contract's writer role, and hosts the directive
	// that names its partners.
	//testkit:contract persister role=writer reader=Get
	Put(ctx context.Context, v Value) error

	// Get is the persister contract's reader role.
	Get(ctx context.Context, key string) (Value, error)
}
