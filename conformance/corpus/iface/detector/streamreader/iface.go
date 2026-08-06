// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package streamreader is the detector-axis fixture for the streamreader shape:
// values yielded lazily as an iter.Seq2 carrying a per-element error.
// Consumers may stop early, so the implementation must not assume the sequence is drained.
//
// No directive appears here: the classification comes from the signature
// alone, so a detector that misfires shows up as wrong generated output
// rather than as a directive that was misread.
package streamreader

import (
	"context"
	"iter"
)

// Value is the element type the stream yields.
type Value struct{ Key, Body string }

// StreamReader is the fixture interface.
//
//testkit:out streamreadertest/ pkg=streamreadertest
//testkit:stub
type StreamReader interface {
	List(ctx context.Context) iter.Seq2[Value, error]
}
