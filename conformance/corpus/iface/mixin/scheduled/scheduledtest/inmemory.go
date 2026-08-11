// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package scheduledtest holds the generated harness and double for
// [go.thesmos.sh/testkit/conformance/corpus/iface/mixin/scheduled], and the
// in-memory subject they are run against — scaffolding for the run, so it lives
// beside the harness rather than in the package stating the shape.
package scheduledtest

import (
	"context"
	"errors"
	"sync"
	"time"

	"go.thesmos.sh/testkit/clock"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/scheduled"
)

// InMemory is the implementation the generated conformance harness is run
// against.
type InMemory struct {
	mu    sync.Mutex
	clk   clock.Clock
	tasks []time.Time
}

var _ scheduled.Mixed = (*InMemory)(nil)

// NewInMemory returns a scheduler on a clock that only moves when told to.
func NewInMemory() *InMemory {
	return &InMemory{clk: clock.NewTestClock(time.Unix(0, 0).UTC())}
}

// At registers a task for the given offset from the clock's current reading.
func (s *InMemory) At(ctx context.Context, after time.Duration) error {
	if err := contextErr(ctx); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.tasks = append(s.tasks, s.clk.Now().Add(after))
	return nil
}

// Fired counts the tasks whose instant the clock has reached.
//
// Computed from the clock rather than tracked by a goroutine: a scheduler
// that fired on a timer would make the count depend on when it was asked,
// and the claim is about the clock rather than about scheduling latency.
func (s *InMemory) Fired(ctx context.Context) (int, error) {
	if err := contextErr(ctx); err != nil {
		return 0, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	now := s.clk.Now()
	var n int
	for _, at := range s.tasks {
		if !at.After(now) {
			n++
		}
	}
	return n, nil
}

// contextErr reports a cancelled or expired context, and tolerates a nil one.
func contextErr(ctx context.Context) error {
	if ctx == nil {
		return errors.New("scheduledtest: nil context")
	}
	return ctx.Err()
}
