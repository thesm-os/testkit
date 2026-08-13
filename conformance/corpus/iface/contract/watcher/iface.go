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
	"time"
)

// Value is the payload the contract's roles carry.
type Value struct{ Key, Body string }

// Subscription is the handle Watch answers — the members the directive's
// next= and stop= name are declared here, on the subscription, because
// reading the next event and ending the watch are the handle's operations,
// not the store's.
type Subscription interface {
	// Next answers the next change within the wait, or reports that none
	// arrived.
	Next(timeout time.Duration) (Value, bool)

	// Stop ends the subscription.
	Stop()
}

// Contract is the fixture interface.
//
//testkit:out watchertest/ pkg=watchertest
//testkit:stub
//testkit:suite
//testkit:model
type Contract interface {
	// Watch is the watcher contract's watch role, and hosts the directive
	// that names its partners — the trigger beside it, and the two members
	// of the subscription it answers.
	//testkit:contract watcher role=watch trigger=Trigger next=Next stop=Stop
	Watch(ctx context.Context, key string) (Subscription, error)

	// Trigger is the watcher contract's trigger role. It carries the value
	// the change publishes, because a watch that fires is only half the
	// claim — what arrived has to be what was written.
	Trigger(ctx context.Context, key string, v Value) error
}
