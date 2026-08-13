// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package singleflight is the contract-axis fixture for the singleflight contract:
// concurrent calls for one key sharing a single computation.
//
// Every role the contract declares is present, because a contract is a
// multi-callable protocol and a fixture missing a partner exercises the
// validator's failure path rather than the contract itself.
package singleflight

import (
	"context"
)

// Value is the payload the contract's roles carry.
type Value struct{ Key, Body string }

// Contract is the fixture interface.
//
//testkit:out singleflighttest/ pkg=singleflighttest
//testkit:stub
//testkit:suite
//testkit:model
type Contract interface {
	// Run is the singleflight contract's fn role, and hosts the directive
	// that names its partners. It takes the computation it deduplicates —
	// the shape a coalescing claim needs, because a run that computes
	// nothing observable cannot show how often it computed.
	//testkit:contract singleflight role=fn
	Run(ctx context.Context, key string, compute func() string) (string, error)

	// Flights reports how many computations have ever run — the observable
	// a coalescing claim counts. It is also what keeps the sequences
	// driving: the run role takes a callable no pool can draw, so this is
	// the method the twin comparison exercises between law rounds.
	Flights(ctx context.Context) (int, error)
}
