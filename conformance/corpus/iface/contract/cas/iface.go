// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package cas is the contract-axis fixture for the cas contract:
// a compare-and-swap write guarded by a version.
//
// Every role the contract declares is present, because a contract is a
// multi-callable protocol and a fixture missing a partner exercises the
// validator's failure path rather than the contract itself.
package cas

import (
	"context"
)

// Value is the payload the contract's roles carry.
type Value struct{ Key, Body string }

// Contract is the fixture interface.
//
//testkit:out castest/ pkg=castest
//testkit:stub
//testkit:suite
//testkit:model
type Contract interface {
	// Put is the cas contract's writer role, and hosts the directive
	// that names its partners.
	//testkit:contract cas role=writer version=Version
	Put(ctx context.Context, v Value) error
}
