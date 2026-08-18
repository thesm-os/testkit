// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package cursoropen is the contract-axis fixture for the cursor
// contract's producer arm: the directive sits on the method that
// answers a fresh cursor, and its next= and close= name members of the
// handle it answers — the produced type declares no directives of its
// own, so the producing method is the only place its contract can be
// licensed. The standalone arm keeps its own fixture in ../cursor.
package cursoropen

import (
	"context"
	"errors"
)

// ErrClosed is what Next reports once Close has run — the contract's
// own sentinel, declared beside the shape so the directive can name it.
var ErrClosed = errors.New("cursoropen: closed")

// Value is the payload the produced cursor drains.
type Value struct{ Key, Body string }

// Cursor is the handle Open answers — the members the directive's
// next= and close= name are declared here, on the cursor, because
// draining and releasing are the handle's operations, not the
// producer's.
type Cursor interface {
	// Next answers the next value plus true, or the zero value plus
	// false when exhausted. After Close, Next reports ErrClosed.
	Next(ctx context.Context) (Value, bool, error)

	// Close releases the cursor. A second Close changes nothing.
	Close(ctx context.Context) error
}

// Contract is the fixture interface.
//
// No //testkit:model: a producer-only interface has no method that
// maps to a sequence action — the produced cursor's laws instantiate
// at the CURSOR's type and lift onto the producer, which is the
// produced-secondary lowering the next-generation model plugin owns.
// The vocabulary is canon now; the lowering arrives with that plugin.
//
//testkit:out cursoropentest/ pkg=cursoropentest
//testkit:stub
//testkit:suite
type Contract interface {
	// Open is the cursor contract's open role, and hosts the directive
	// that names the handle's members and the sentinel they share.
	//testkit:contract cursor role=open next=Next close=Close sentinel=ErrClosed
	Open(ctx context.Context) (Cursor, error)
}
