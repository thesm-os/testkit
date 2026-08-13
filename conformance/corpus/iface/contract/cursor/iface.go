// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package cursor is the contract-axis fixture for the cursor contract:
// a cursor drained by Next and released by Close.
//
// Every role the contract declares is present, because a contract is a
// multi-callable protocol and a fixture missing a partner exercises the
// validator's failure path rather than the contract itself.
package cursor

import (
	"context"
	"errors"
)

// ErrClosed is what Next reports once Close has run — the contract's own
// sentinel, declared beside the shape so the directive can name it.
var ErrClosed = errors.New("cursor: closed")

// Value is the payload the contract's roles carry.
type Value struct{ Key, Body string }

// Contract is the fixture interface.
//
//testkit:out cursortest/ pkg=cursortest
//testkit:stub
//testkit:suite
//testkit:model
type Contract interface {
	// Next is the cursor contract's next role, and hosts the directive
	// that names its partners.
	//testkit:contract cursor role=next close=Close sentinel=ErrClosed
	Next(ctx context.Context) (Value, bool, error)

	// Close is the cursor contract's close role.
	Close(ctx context.Context) error
}
