// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package ifmatch is the contract-axis fixture for the if-match contract:
// a write conditional on a predicate.
//
// Every role the contract declares is present, because a contract is a
// multi-callable protocol and a fixture missing a partner exercises the
// validator's failure path rather than the contract itself.
package ifmatch

import (
	"context"
)

// Value is the payload the contract's roles carry.
type Value struct{ Key, Body string }

// Contract is the fixture interface.
//
//testkit:out ifmatchtest/ pkg=ifmatchtest
//testkit:stub
type Contract interface {
	// Put is the if-match contract's writer role, and hosts the directive
	// that names its partners.
	//testkit:contract if-match role=writer pred=Match
	Put(ctx context.Context, v Value) error
}
