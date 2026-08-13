// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package publisherexactlyonce is the contract-axis fixture for the publisher contract's
// exactly-once bound: each subscriber receives a published message exactly once — neither loss nor duplication.
//
// The mode rides the directive, and the bound law it selects counts each
// subscriber's copies of a published message against it. The redeliver role
// is armed here through Replay, whose dedupe is the exactly-once claim
// itself: the law re-offers the published message and the duplicate must
// never reach a subscriber — which is what tells this mode apart from the
// at-least-once fixture that redelivers without asking.
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
	// that names its partners — the redeliver role included.
	//testkit:contract publisher role=publish subscribe=Subscribe redeliver=Replay mode=exactly-once
	Publish(ctx context.Context, v Value) error

	// Replay is the publisher contract's redeliver role: it re-offers a
	// message the broker may already have delivered, and the exactly-once
	// claim is that the duplicate never reaches a subscriber.
	Replay(ctx context.Context, v Value) error

	// Subscribe is the publisher contract's subscribe role.
	Subscribe(ctx context.Context) (<-chan Value, error)
}
