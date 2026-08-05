// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package leaderelection is the contract-axis fixture for the leader-election contract:
// a leadership protocol.
//
// Every role the contract declares is present, because a contract is a
// multi-callable protocol and a fixture missing a partner exercises the
// validator's failure path rather than the contract itself.
package leaderelection

import (
	"context"
)

// Value is the payload the contract's roles carry.
type Value struct{ Key, Body string }

// Contract is the fixture interface.
type Contract interface {
	// Campaign is the leader-election contract's campaign role, and hosts the directive
	// that names its partners.
	//testkit:contract leader-election role=campaign resign=Resign isleader=IsLeader
	Campaign(ctx context.Context) error

	// Resign is the leader-election contract's resign role.
	Resign(ctx context.Context) error

	// IsLeader is the leader-election contract's isleader role.
	IsLeader(ctx context.Context) bool
}
