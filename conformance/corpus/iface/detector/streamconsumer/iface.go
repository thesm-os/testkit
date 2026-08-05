// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package streamconsumer is the detector-axis fixture for the streamconsumer shape:
// a sequence in rather than out — the mirror of a stream reader. The
// interesting case is a caller whose sequence stops short.
//
// No directive appears here: the classification comes from the signature
// alone, so a detector that misfires shows up as wrong generated output
// rather than as a directive that was misread.
package streamconsumer

import (
	"context"
	"iter"
)

// Value is the element type the fixture ingests.
type Value struct{ Key, Body string }

// StreamConsumer is the fixture interface.
type StreamConsumer interface {
	Ingest(ctx context.Context, src iter.Seq[Value]) (int, error)
}
