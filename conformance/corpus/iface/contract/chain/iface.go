// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package chain is the contract-axis fixture for the chain contract: entries
// appended to a log, replayed in order, and verified against the digest that
// links them.
//
// Every role the contract declares is present, because a contract is a
// multi-callable protocol and a fixture missing a partner exercises the
// validator's failure path rather than the contract itself.
package chain

import (
	"context"
)

// Entry is one link in the chain.
type Entry struct{ Key, Body string }

// Contract is the fixture interface.
//
//testkit:out chaintest/ pkg=chaintest
//testkit:stub
//testkit:suite
//testkit:model
type Contract interface {
	// Append is the chain contract's append role, and hosts the directive
	// that names its partner. The contract requires a replay beside every
	// append: a log nothing can read back states nothing.
	//testkit:contract chain role=append replay=Replay
	Append(ctx context.Context, e Entry) error

	// Replay is the chain contract's replay role.
	Replay(ctx context.Context) ([]Entry, error)

	// Verify is the chain contract's verify role: it recomputes the links
	// and reports a break rather than serving a broken chain.
	//testkit:contract chain role=verify
	Verify(ctx context.Context) error
}
