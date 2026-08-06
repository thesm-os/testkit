// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package circuitbreaker is the contract-axis fixture for the circuit-breaker contract:
// a call that trips open after repeated failure.
//
// Every role the contract declares is present, because a contract is a
// multi-callable protocol and a fixture missing a partner exercises the
// validator's failure path rather than the contract itself.
package circuitbreaker

import (
	"context"
)

// Value is the payload the contract's roles carry.
type Value struct{ Key, Body string }

// Contract is the fixture interface.
//
//testkit:out circuitbreakertest/ pkg=circuitbreakertest
//testkit:stub
type Contract interface {
	// Run is the circuit-breaker contract's fn role, and hosts the directive
	// that names its partners.
	//testkit:contract circuit-breaker role=fn
	Run(ctx context.Context, key string) error
}
