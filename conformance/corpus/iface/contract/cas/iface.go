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
	"errors"
)

// ErrMismatch is what Put reports for a stale version — the contract's own
// sentinel, declared beside the shape so the directive can name it.
var ErrMismatch = errors.New("cas: the write's version is stale")

// Value is the payload the contract's roles carry: the body, and the version
// the compare-and-set guards it by.
type Value struct {
	Body    string
	Version int64
}

// Contract is the fixture interface.
//
//testkit:out castest/ pkg=castest
//testkit:stub
//testkit:suite
//testkit:model
type Contract interface {
	// Put is the cas contract's writer role, and hosts the directive
	// that names its partners.
	//testkit:contract cas role=writer version=Version mismatch=ErrMismatch
	Put(ctx context.Context, v Value) error

	// Get answers what the cell holds — the observation a compare-and-set
	// is meaningless without.
	Get(ctx context.Context) (Value, error)
}
