// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package publisherexactlyonce is the contract-axis fixture for the publisher contract's
// exactly-once bound: each subscriber receives a published message exactly once — neither loss nor duplication.
//
// The mode rides the directive, and the bound law it selects counts each
// subscriber's copies of a published message against it. The redeliver role
// stays undeclared here, so the bound is exercised on the single publish —
// the redelivery arm waits on a fixture that declares the role.
package publisherexactlyonce

import (
	"context"
)

// Value is the payload the contract's roles carry.
type Value struct{ Key, Body string }

// Contract is the fixture interface.
//
//testkit:out publisherexactlyoncetest/ pkg=publisherexactlyoncetest
//testkit:stub
//testkit:suite
//testkit:model
type Contract interface {
	// Publish is the publisher contract's publish role, and hosts the directive
	// that names its partners.
	//testkit:contract publisher role=publish subscribe=Subscribe mode=exactly-once
	Publish(ctx context.Context, v Value) error

	// Subscribe is the publisher contract's subscribe role.
	Subscribe(ctx context.Context) (<-chan Value, error)
}
