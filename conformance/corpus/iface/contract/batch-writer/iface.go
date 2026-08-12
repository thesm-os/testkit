// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package batchwriter is the contract-axis fixture for the batch-writer contract:
// a write that takes many values at once.
//
// Every role the contract declares is present, because a contract is a
// multi-callable protocol and a fixture missing a partner exercises the
// validator's failure path rather than the contract itself.
package batchwriter

import (
	"context"
)

// Value is the payload the contract's roles carry.
type Value struct{ Key, Body string }

// Contract is the fixture interface.
//
//testkit:out batchwritertest/ pkg=batchwritertest
//testkit:stub
//testkit:suite
//testkit:model
type Contract interface {
	// Put is the batch-writer contract's writer role, and hosts the directive
	// that names its partners.
	//testkit:contract batch-writer role=writer mode=atomic
	Put(ctx context.Context, v Value) error
}
