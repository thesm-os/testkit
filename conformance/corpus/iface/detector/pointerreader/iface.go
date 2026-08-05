// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package pointerreader is the detector-axis fixture for the pointerreader shape:
// a pointer return, so a miss is expressible as nil rather than as a zero
// value. That makes the nil-safety law meaningful here in a way it is not for a value return.
//
// No directive appears here: the classification comes from the signature
// alone, so a detector that misfires shows up as wrong generated output
// rather than as a directive that was misread.
package pointerreader

import "context"

// Value is the payload the fixture reads.
type Value struct{ Key, Body string }

// PointerReader is the fixture interface.
type PointerReader interface {
	// Find returns exactly one value, and that value is the pointer. Adding an
	// error return takes the method out of this shape entirely — it becomes an
	// ordinary reader, because nil no longer has to carry the miss.
	Find(ctx context.Context, key string) *Value
}
