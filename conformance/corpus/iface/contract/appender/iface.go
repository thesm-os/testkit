// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package appender is the contract-axis fixture for the appender contract:
// an append-only log whose writes only ever add.
//
// Every role the contract declares is present, because a contract is a
// multi-callable protocol and a fixture missing a partner exercises the
// validator's failure path rather than the contract itself.
package appender

import (
	"context"
)

// Value is the payload the contract's roles carry.
type Value struct{ Key, Body string }

// Contract is the fixture interface.
type Contract interface {
	// Run is the appender contract's fn role, and hosts the directive
	// that names its partners.
	//testkit:contract appender role=fn
	Run(ctx context.Context, key string) error
}
