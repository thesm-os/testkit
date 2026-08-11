// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package pagination is the contract-axis fixture for the pagination contract:
// a reader whose protocol cursors through pages.
//
// Every role the contract declares is present, because a contract is a
// multi-callable protocol and a fixture missing a partner exercises the
// validator's failure path rather than the contract itself.
package pagination

import (
	"context"
)

// Value is the payload the contract's roles carry.
type Value struct{ Key, Body string }

// Contract is the fixture interface.
//
//testkit:out paginationtest/ pkg=paginationtest
//testkit:stub
//testkit:suite
type Contract interface {
	// Get is the pagination contract's reader role, and hosts the directive
	// that names its partners.
	//testkit:contract pagination role=reader cursor=Cursor
	Get(ctx context.Context, key string) (Value, error)
}
