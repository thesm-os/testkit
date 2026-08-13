// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package publisheratleastonce is the contract-axis fixture for the publisher contract's
// at-least-once bound: each subscriber receives a published message one or more times — duplicates permitted, loss forbidden.
//
// The mode rides the directive, and the bound law it selects counts each
// subscriber's copies of a published message against it. The redeliver role
// stays undeclared here, so the bound is exercised on the single publish and
// the header says the arm went unarmed — `publisher-redeliver` and the
// exactly-once sibling are the fixtures that declare the role.
package publisheratleastonce

import (
	"context"
)

// Value is the payload the contract's roles carry.
type Value struct{ Key, Body string }

// Contract is the fixture interface.
//
//testkit:out publisheratleastoncetest/ pkg=publisheratleastoncetest
//testkit:stub
//testkit:suite
//testkit:model
type Contract interface {
	// Publish is the publisher contract's publish role, and hosts the directive
	// that names its partners.
	//testkit:contract publisher role=publish subscribe=Subscribe mode=at-least-once
	Publish(ctx context.Context, v Value) error

	// Subscribe is the publisher contract's subscribe role.
	Subscribe(ctx context.Context) (<-chan Value, error)
}
