// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package ifabsent is the contract-axis fixture for the if-absent contract:
// a write that only lands when the key is absent.
//
// Every role the contract declares is present, because a contract is a
// multi-callable protocol and a fixture missing a partner exercises the
// validator's failure path rather than the contract itself.
package ifabsent

import (
	"context"
)

// Value is the payload the contract's roles carry.
type Value struct{ Key, Body string }

// Contract is the fixture interface.
//
//testkit:out ifabsenttest/ pkg=ifabsenttest
//testkit:stub
type Contract interface {
	// Put is the if-absent contract's writer role, and hosts the directive
	// that names its partners.
	//testkit:contract if-absent role=writer
	Put(ctx context.Context, v Value) error
}
