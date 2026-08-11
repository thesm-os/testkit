// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package tx is the contract-axis fixture for the tx contract:
// a begin/commit/rollback triple.
//
// Every role the contract declares is present, because a contract is a
// multi-callable protocol and a fixture missing a partner exercises the
// validator's failure path rather than the contract itself.
package tx

import (
	"context"
)

// Value is the payload the contract's roles carry.
type Value struct{ Key, Body string }

// Contract is the fixture interface.
//
//testkit:out txtest/ pkg=txtest
//testkit:stub
//testkit:suite
type Contract interface {
	// Begin is the tx contract's begin role, and hosts the directive
	// that names its partners.
	//testkit:contract tx role=begin commit=Commit rollback=Rollback
	Begin(ctx context.Context) error

	// Commit is the tx contract's commit role.
	Commit(ctx context.Context) error

	// Rollback is the tx contract's rollback role.
	Rollback(ctx context.Context) error
}
