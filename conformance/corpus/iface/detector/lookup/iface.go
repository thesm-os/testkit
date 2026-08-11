// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package lookup is the detector-axis fixture for the lookup shape:
// two values and a presence flag. The second return is what separates it
// from a bool-returning reader: metadata arrives with the value, not from a second call.
//
// No directive appears here: the classification comes from the signature
// alone, so a detector that misfires shows up as wrong generated output
// rather than as a directive that was misread.
package lookup

// Value is the primary return.
type Value struct{ Key, Body string }

// Meta is the secondary return.
type Meta struct{ Revision int }

// Lookup is the fixture interface.
//
//testkit:out lookuptest/ pkg=lookuptest
//testkit:stub
//testkit:suite
type Lookup interface {
	Inspect(key string) (Value, Meta, bool)
}
