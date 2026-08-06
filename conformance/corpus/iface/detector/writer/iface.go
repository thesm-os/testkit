// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package writer is the detector-axis fixture for the writer shape:
// a value in, an error out. The absence of a returned value separates it
// from a composite writer and from a mutator.
//
// No directive appears here: the classification comes from the signature
// alone, so a detector that misfires shows up as wrong generated output
// rather than as a directive that was misread.
package writer

import (
	"context"
)

// Value is the payload the fixture stores.
type Value struct{ Key, Body string }

// Writer is the fixture interface.
//
//testkit:out writertest/ pkg=writertest
//testkit:stub
type Writer interface {
	Put(ctx context.Context, v Value) error
}
