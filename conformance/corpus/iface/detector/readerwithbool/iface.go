// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package readerwithbool is the detector-axis fixture for the readerwithbool shape:
// the map-style fetch: presence rides in the boolean rather than in an
// error, so a miss is an ordinary outcome instead of a failure.
//
// No directive appears here: the classification comes from the signature
// alone, so a detector that misfires shows up as wrong generated output
// rather than as a directive that was misread.
package readerwithbool

import (
	"context"
)

// Value is the payload the fixture reads.
type Value struct{ Key, Body string }

// ReaderWithBool is the fixture interface.
//
//testkit:out readerwithbooltest/ pkg=readerwithbooltest
//testkit:stub
type ReaderWithBool interface {
	Load(ctx context.Context, key string) (Value, bool)
}
