// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package workflow is the contract-axis fixture for the workflow contract:
// a callable driving a state machine.
//
// Every role the contract declares is present, because a contract is a
// multi-callable protocol and a fixture missing a partner exercises the
// validator's failure path rather than the contract itself.
package workflow

import (
	"context"
)

// Value is the payload the contract's roles carry.
type Value struct{ Key, Body string }

// Contract is the fixture interface.
//
//testkit:out workflowtest/ pkg=workflowtest
//testkit:stub
//testkit:suite
//testkit:model
type Contract interface {
	// Run is the workflow contract's fn role, and hosts the directive
	// that names its partners.
	//testkit:contract workflow role=fn transitions=Draft>Live
	Run(ctx context.Context, key string) error
}
