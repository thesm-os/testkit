// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package roledtypes is the language-axis fixture for a roled BARE
// PARAMETER: a method taking a named type directly rather than a request
// struct.
//
// The shape decides where a role can be written. A request struct carries
// the stamp on the field holding the value; a bare parameter has no field,
// so the only declaration left is the named type itself. Both arms exist
// in this fixture, so the pool derivation is exercised by the gate rather
// than by the validated packs alone — which is where it lived, untested,
// while every corpus interface derived an empty pool set.
package roledtypes

import "context"

// Key identifies one entry. A named type rather than a bare string, and
// the role lives ON the type: Put and Get take Key directly, so there is
// no field to carry the stamp.
//
//testkit:role key
//testkit:default "test-key"
type Key string

// Payload is what an entry holds — the request-struct arm, whose role
// sits on the field.
type Payload struct {
	//testkit:role payload
	//testkit:default "test-body"
	Body string
}

// Store is the fixture interface.
//
//testkit:out roledtypestest/ pkg=roledtypestest
//testkit:stub
//testkit:suite
type Store interface {
	// Put writes a payload under a key, drawing both roles.
	Put(ctx context.Context, key Key, payload Payload) error

	// Get reads one back, drawing the key alone.
	Get(ctx context.Context, key Key) (Payload, error)
}
