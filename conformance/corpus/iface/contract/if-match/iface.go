// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package ifmatch is the contract-axis fixture for the if-match contract:
// a write conditional on a predicate.
//
// Every role the contract declares is present, because a contract is a
// multi-callable protocol and a fixture missing a partner exercises the
// validator's failure path rather than the contract itself.
//
// The predicate has two spellings and this fixture uses the callable one.
// `match=Match` names a method the resolver qualifies, reports when it names
// nothing in scope, and back-stamps onto the predicate. `pred=` is the other
// form and carries an expression — `pred=Version==Expected` — which the
// resolver deliberately leaves verbatim, because there is no callable in it.
//
// A conformance check has to call the predicate, so only the callable form is
// reachable from this tier. The expression form is a declaration the model tier
// can act on and this one cannot.
package ifmatch

import (
	"context"
)

// Value is the payload the contract's roles carry.
type Value struct{ Key, Body string }

// Contract is the fixture interface.
//
//testkit:out ifmatchtest/ pkg=ifmatchtest
//testkit:stub
//testkit:suite
type Contract interface {
	// Put is the if-match contract's writer role, and hosts the directive
	// that names its partners.
	//testkit:contract if-match role=writer match=Match
	Put(ctx context.Context, v Value) error

	// Match is the predicate the write is conditional on. It answers about the
	// value Put takes, because a predicate over anything else is one the write
	// cannot be conditional on.
	Match(ctx context.Context, v Value) (bool, error)
}
