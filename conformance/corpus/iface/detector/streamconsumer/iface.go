// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package streamconsumer is the detector-axis fixture for the streamconsumer shape:
// a stream in rather than out.
//
// It is not quite the mirror of a stream reader, and the asymmetry is worth
// knowing: eidos recognises a produced stream as an iter.Seq2 return, but a
// consumed stream as an *interface* parameter. A method taking iter.Seq is a
// method taking a func, and no amount of naming makes the detector see a
// stream in it.
//
// No directive appears here: the classification comes from the signature
// alone, so a detector that misfires shows up as wrong generated output
// rather than as a directive that was misread.
package streamconsumer

import "context"

// Value is the element type the fixture ingests.
type Value struct{ Key, Body string }

// Source is the stream Ingest consumes. It has to be an interface: the
// detector tests whether the parameter is one, and an iter.Seq is a func type
// however stream-like it reads.
//
//testkit:out streamconsumertest/ pkg=streamconsumertest
//testkit:stub
type Source interface {
	Next(ctx context.Context) (Value, bool, error)
}

// StreamConsumer is the fixture interface.
//
//testkit:out streamconsumertest/ pkg=streamconsumertest
//testkit:stub
type StreamConsumer interface {
	// Ingest takes one non-context parameter and returns one value plus an
	// error. Both counts are load-bearing: a second parameter or a second
	// value takes the method out of the shape.
	Ingest(ctx context.Context, src Source) (int, error)
}
