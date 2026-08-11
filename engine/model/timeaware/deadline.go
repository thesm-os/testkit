// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

//nolint:errorlint // diagnostic, not wrapping
package timeaware

import (
	"context"
	"fmt"
	"time"

	"pgregory.net/rapid"

	"go.thesmos.sh/testkit/core/lawid"
)

// DeadlineRespecting verifies an operation invoked with a
// deadline-bearing context returns once the deadline fires.
//
// The checker invokes Op in a goroutine with a child context that
// carries a deadline of cfg.Deadline from now, then advances the
// test clock past the deadline and asserts Op returned via the
// supplied done channel. A failure indicates the SUT either
// ignores the context deadline or never checks it.
type DeadlineRespecting[T any] struct {
	// Op is the operation under test. It must respect the context
	// deadline. The implementation receives the ctx and SUT; it
	// returns when the deadline fires or when the operation
	// completes naturally.
	Op func(ctx context.Context, sut T) error

	// Deadline is the relative deadline applied to Op's context.
	Deadline time.Duration

	// Advance advances the test clock by the supplied duration.
	Advance func(time.Duration)

	// AwaitFor is the upper bound on how long the law waits for
	// Op to return after the clock advance. Zero defaults to 1
	// second; the law uses real-time sleep here so a misbehaving
	// SUT can't hang the property suite indefinitely.
	AwaitFor time.Duration
}

// ID returns the stable identifier for this law.
func (DeadlineRespecting[T]) ID() string { return lawid.DeadlineRespecting }

// REQID returns an empty string (auto-derived).
func (DeadlineRespecting[T]) REQID() string { return "" }

// Check verifies Op returns after the deadline advance.
func (l DeadlineRespecting[T]) Check(_ *rapid.T, sut, _ T) error {
	ctx, cancel := context.WithTimeout(context.Background(), l.Deadline)
	defer cancel()

	done := make(chan error, 1)
	go func() {
		done <- l.Op(ctx, sut)
	}()

	// Advance the test clock past the deadline.
	l.Advance(l.Deadline + time.Millisecond)

	wait := l.AwaitFor
	if wait <= 0 {
		wait = time.Second
	}
	select {
	case <-done:
		return nil
	case <-time.After(wait):
		return fmt.Errorf("deadline-respecting law: Op did not return within %v of deadline advance", wait)
	}
}
