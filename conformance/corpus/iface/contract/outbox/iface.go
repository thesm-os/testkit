// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package outbox is the contract-axis fixture for the outbox contract:
// a transactional outbox and its reader.
//
// Every role the contract declares is present, because a contract is a
// multi-callable protocol and a fixture missing a partner exercises the
// validator's failure path rather than the contract itself.
package outbox

import (
	"context"
)

// Value is the payload the contract's roles carry.
type Value struct{ Key, Body string }

// Contract is the fixture interface.
//
//testkit:out outboxtest/ pkg=outboxtest
//testkit:stub
//testkit:suite
//testkit:model
type Contract interface {
	// Append is the outbox contract's append role, and hosts the directive
	// that names its partners.
	//testkit:contract outbox role=append subscribe=Subscribe
	Append(ctx context.Context, v Value) error

	// Subscribe is the outbox contract's subscribe role.
	Subscribe(ctx context.Context) (<-chan Value, error)
}
