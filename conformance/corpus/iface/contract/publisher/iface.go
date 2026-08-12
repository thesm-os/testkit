// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package publisher is the contract-axis fixture for the publisher contract:
// a publish paired with the subscribe that delivers it.
//
// Every role the contract declares is present, because a contract is a
// multi-callable protocol and a fixture missing a partner exercises the
// validator's failure path rather than the contract itself.
package publisher

import (
	"context"
)

// Value is the payload the contract's roles carry.
type Value struct{ Key, Body string }

// Contract is the fixture interface.
//
//testkit:out publishertest/ pkg=publishertest
//testkit:stub
//testkit:suite
//testkit:model
type Contract interface {
	// Publish is the publisher contract's publish role, and hosts the directive
	// that names its partners.
	//testkit:contract publisher role=publish subscribe=Subscribe
	Publish(ctx context.Context, v Value) error

	// Subscribe is the publisher contract's subscribe role.
	Subscribe(ctx context.Context) (<-chan Value, error)
}
