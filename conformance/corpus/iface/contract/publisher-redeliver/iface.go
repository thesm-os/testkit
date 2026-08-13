// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package publisherredeliver is the contract-axis fixture that arms the
// publisher's redeliver role: the at-least-once bound proven through an
// actual redelivery rather than on the single publish.
//
// Republish re-offers a message the broker already delivered, so each
// subscriber sees the duplicate the mode permits — which is exactly what
// tells at-least-once apart from exactly-once. Its unarmed siblings
// exercise the role's optional omission, and their headers say so.
package publisherredeliver

import (
	"context"
)

// Value is the payload the contract's roles carry.
type Value struct{ Key, Body string }

// Contract is the fixture interface.
//
//testkit:out publisherredelivertest/ pkg=publisherredelivertest
//testkit:stub
//testkit:suite
//testkit:model
type Contract interface {
	// Publish is the publisher contract's publish role, and hosts the
	// directive that names its partners — the redeliver role included.
	//testkit:contract publisher role=publish subscribe=Subscribe redeliver=Republish mode=at-least-once
	Publish(ctx context.Context, v Value) error

	// Republish is the publisher contract's redeliver role: it re-offers a
	// message to every current subscriber, the duplicate this mode permits.
	Republish(ctx context.Context, v Value) error

	// Subscribe is the publisher contract's subscribe role.
	Subscribe(ctx context.Context) (<-chan Value, error)
}
