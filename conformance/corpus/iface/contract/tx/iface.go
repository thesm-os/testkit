// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package tx is the contract-axis fixture for the tx contract:
// a begin/commit/rollback triple threading one handle.
//
// Every role the contract declares is present, because a contract is a
// multi-callable protocol and a fixture missing a partner exercises the
// validator's failure path rather than the contract itself.
package tx

import (
	"context"
	"errors"
)

// Value is the payload the contract's roles carry.
type Value struct{ Key, Body string }

// Tx is the handle Begin answers and both terminal operations consume.
type Tx struct{ ID int64 }

// ErrTxClosed is what a terminal operation reports on a transaction the
// other terminal operation already closed.
var ErrTxClosed = errors.New("tx: closed")

// ErrNotFound is what Get reports for a key no transaction committed.
var ErrNotFound = errors.New("tx: not found")

// Contract is the fixture interface.
//
//testkit:out txtest/ pkg=txtest
//testkit:stub
//testkit:suite
//testkit:model
type Contract interface {
	// Begin opens a transaction and answers the handle its terminal pair
	// threads — the shape a commit-XOR-rollback claim needs, because a
	// mutex over an unnamed transaction is unobservable.
	//testkit:contract tx role=begin commit=Commit rollback=Rollback closed=ErrTxClosed
	Begin(ctx context.Context) (Tx, error)

	// Commit terminally applies the handle's transaction.
	Commit(ctx context.Context, tx Tx) error

	// Rollback terminally discards the handle's transaction.
	Rollback(ctx context.Context, tx Tx) error

	// Get observes the committed state — what an open transaction staged
	// must not have touched, which is the mid-transaction claim's outside
	// read.
	Get(ctx context.Context, key string) (Value, error)
}
