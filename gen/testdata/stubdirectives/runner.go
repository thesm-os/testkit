// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package newdirectives exercises the retry-succeeds-on-attempt, partition,
// and order-after directives.
package stubdirectives

import (
	"context"
	"errors"
)

//go:generate testkit stub -o runnertest/runner_stub.gen.go Runner

// ErrUnavailable is returned when the service is temporarily unavailable.
var ErrUnavailable = errors.New("unavailable")

// AppendRequest is a request to append data.
type AppendRequest struct {
	RunID string
	Data  []byte
}

// AppendResult is the result of an append operation.
type AppendResult struct {
	Offset int
}

// Runner exercises the new directive catalog.
type Runner interface {
	//testkit:order-after Open
	//testkit:retryable
	//testkit:retry-succeeds-on-attempt 3
	//testkit:partition RunID
	// Append appends data to a run.
	Append(ctx context.Context, req AppendRequest) (AppendResult, error)

	// Open opens a run. Must be called before Append.
	Open(ctx context.Context, runID string) error

	// Close closes a run.
	Close(ctx context.Context, runID string) error
}
