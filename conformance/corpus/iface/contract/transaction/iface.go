// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package transaction is the contract-axis fixture for the transaction contract:
// a callable running inside a transactional scope.
//
// Every role the contract declares is present, because a contract is a
// multi-callable protocol and a fixture missing a partner exercises the
// validator's failure path rather than the contract itself.
package transaction

import (
	"context"
	"errors"
)

// Value is the payload the contract's roles carry.
type Value struct{ Key, Body string }

// ErrNotFound is what Get reports for a key no transaction committed.
var ErrNotFound = errors.New("transaction: not found")

// Contract is the fixture interface.
//
//testkit:out transactiontest/ pkg=transactiontest
//testkit:stub
//testkit:suite
//testkit:model
type Contract interface {
	// Run is the transaction contract's fn role, and hosts the directive
	// that names its partners. It accepts the body it scopes — the shape a
	// rollback claim needs, because the claim is about what an erroring
	// body leaves behind, and a run that takes no body cannot be made to
	// fail on demand.
	//testkit:contract transaction role=fn notfound=ErrNotFound
	Run(ctx context.Context, body func(ctx context.Context) error) error

	// Put is the write a body performs inside the scope Run opened, and the
	// reason the rollback claim is checkable at all: a claim about what an
	// erroring body leaves behind needs the body to leave something behind.
	//
	// Called outside a run it writes through, which is the ordinary store
	// behaviour a keyed writer promises; called inside one it stages, and an
	// erroring body discards the staging whole.
	Put(ctx context.Context, key string, v Value) error

	// Get observes the committed state a rolled-back run must not have
	// touched.
	Get(ctx context.Context, key string) (Value, error)
}
