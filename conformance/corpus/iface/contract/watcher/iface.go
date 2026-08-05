// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package watcher is the contract-axis fixture for the watcher contract:
// a watch woken by a trigger.
//
// Every role the contract declares is present, because a contract is a
// multi-callable protocol and a fixture missing a partner exercises the
// validator's failure path rather than the contract itself.
package watcher

import (
	"context"
)

// Value is the payload the contract's roles carry.
type Value struct{ Key, Body string }

// Contract is the fixture interface.
type Contract interface {
	// Watch is the watcher contract's watch role, and hosts the directive
	// that names its partners.
	//testkit:contract watcher role=watch trigger=Trigger
	Watch(ctx context.Context, key string) (<-chan Value, error)

	// Trigger is the watcher contract's trigger role.
	Trigger(ctx context.Context, key string) error
}
