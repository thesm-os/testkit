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
)

// Value is the payload the contract's roles carry.
type Value struct{ Key, Body string }

// Contract is the fixture interface.
//
//testkit:out transactiontest/ pkg=transactiontest
//testkit:stub
//testkit:suite
//testkit:model
type Contract interface {
	// Run is the transaction contract's fn role, and hosts the directive
	// that names its partners.
	//testkit:contract transaction role=fn
	Run(ctx context.Context, key string) error
}
