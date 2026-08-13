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
//testkit:suite
//testkit:model
type StreamReader interface {
	// Add gives the stream something to yield. Without a writer the subject
	// stays empty forever, and every claim about what a drain returns holds
	// by vacancy — two drains of nothing agree, a drain of nothing
	// terminates, and no defect worn on List can make either false.
	Add(ctx context.Context, v Value) error

	List(ctx context.Context) iter.Seq2[Value, error]
}
