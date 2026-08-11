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
type Contract interface {
	// Run is the singleflight contract's fn role, and hosts the directive
	// that names its partners.
	//testkit:contract singleflight role=fn
	Run(ctx context.Context, key string) error
}
