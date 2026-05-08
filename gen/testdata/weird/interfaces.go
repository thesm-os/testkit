// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package weird exercises the spec generator with interfaces that don't
// fit the typical CRUD/store patterns. These test method shapes the
// codebase actually encounters: codecs, schedulers, validators,
// multi-param methods, void methods, mixed context/no-context,
// and methods returning unusual type combinations.
package weird

import (
	"context"
	"errors"
	"io"
	"iter"
	"time"
)

//go:generate testkit suite -o weirdtest/weird_spec.gen.go Codec
//go:generate testkit bench -o weirdtest/weird_bench.gen.go Codec
//go:generate testkit stub  -o weirdtest/weird_stub.gen.go  Codec
//go:generate testkit suite -o weirdtest/scheduler_spec.gen.go Scheduler
//go:generate testkit bench -o weirdtest/scheduler_bench.gen.go Scheduler
//go:generate testkit stub  -o weirdtest/scheduler_stub.gen.go  Scheduler

// ErrInvalidInput is returned when encoding/decoding fails.
var ErrInvalidInput = errors.New("invalid input")

// ErrNotScheduled is returned when a task is not found.
var ErrNotScheduled = errors.New("not scheduled")

// --- Codec: no context, io.Reader/io.Writer params, []byte returns ---

// Codec exercises interface-typed params, []byte returns, and no context.
type Codec interface {
	// Encode writes v to w. Writer param (interface-typed).
	Encode(w io.Writer, v any) error

	// Decode reads from r. Reader param (interface-typed).
	Decode(r io.Reader, v any) error

	// MarshalBinary returns bytes. Pure-ish but returns error.
	MarshalBinary(v any) ([]byte, error)

	// ContentType returns the MIME type. Truly pure.
	ContentType() string

	// Handles reports whether this codec handles the given MIME type.
	Handles(mime string) bool
}

// --- Scheduler: multi-param methods, time.Duration, channels ---

// Task is a scheduled unit of work.
type Task struct {
	ID       string
	Interval time.Duration
	Fn       func(context.Context) error
}

// TaskStatus holds runtime info about a scheduled task.
type TaskStatus struct {
	ID       string
	Running  bool
	LastRun  time.Time
	RunCount int
}

// Scheduler exercises multi-param methods (falls to Unknown),
// time-typed params, channel-like patterns, and mixed shapes.
type Scheduler interface {
	// Schedule takes multiple non-ctx params — falls to Unknown shape.
	Schedule(ctx context.Context, id string, interval time.Duration, fn func(context.Context) error) error

	// Cancel is Deleter-shaped (with directive).
	//testkit:deleter
	//testkit:errors ErrNotScheduled
	Cancel(ctx context.Context, id string) error

	// Status is Reader-shaped.
	//testkit:errors ErrNotScheduled
	Status(ctx context.Context, id string) (TaskStatus, error)

	// Running is Aggregator-shaped.
	Running(ctx context.Context) (int, error)

	// Flush is Lifecycle-shaped.
	Flush(ctx context.Context) error

	// Tasks returns all scheduled tasks as a stream.
	Tasks(ctx context.Context) iter.Seq2[TaskStatus, error]

	// Name is Pure-shaped.
	Name() string
}
