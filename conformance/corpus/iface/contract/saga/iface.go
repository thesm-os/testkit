// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package saga is the contract-axis fixture for the saga contract:
// a step paired with the compensation that undoes it.
//
// Every role the contract declares is present, because a contract is a
// multi-callable protocol and a fixture missing a partner exercises the
// validator's failure path rather than the contract itself.
package saga

import (
	"context"
)

// Value is the payload the contract's roles carry.
type Value struct{ Key, Body string }

// Contract is the fixture interface.
type Contract interface {
	// Step is the saga contract's step role, and hosts the directive
	// that names its partners.
	//testkit:contract saga role=step compensate=Compensate
	Step(ctx context.Context, v Value) error

	// Compensate is the saga contract's compensate role.
	Compensate(ctx context.Context, v Value) error
}
