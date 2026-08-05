// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package ratelimit is the contract-axis fixture for the rate-limit contract:
// a call bounded by a rate and a burst.
//
// Every role the contract declares is present, because a contract is a
// multi-callable protocol and a fixture missing a partner exercises the
// validator's failure path rather than the contract itself.
package ratelimit

import (
	"context"
)

// Value is the payload the contract's roles carry.
type Value struct{ Key, Body string }

// Contract is the fixture interface.
type Contract interface {
	// Run is the rate-limit contract's fn role, and hosts the directive
	// that names its partners.
	//testkit:contract rate-limit role=fn rate=100 burst=10
	Run(ctx context.Context, key string) error
}
