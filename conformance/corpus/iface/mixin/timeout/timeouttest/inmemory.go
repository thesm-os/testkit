// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package timeouttest holds the generated harness and double for
// [go.thesmos.sh/testkit/conformance/corpus/iface/mixin/timeout], and the
// in-memory subject they are run against — scaffolding for the run, so it lives
// beside the harness rather than in the package stating the shape.
package timeouttest

import (
	"context"
	"errors"
	"sync"
	"time"

	"go.thesmos.sh/testkit/clock"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/timeout"
)

// InMemory is the implementation the generated conformance harness is run
// against.
//
// It takes a clock, which is what makes the generated budget check mean
// something: the harness measures on the clock it holds, so a subject built with
// the same one is measured on the time it means to spend rather than on how
// loaded the machine was. An implementation that took no clock would be timed
// against the real one and would be a flake waiting for a busy CI box.
type InMemory struct {
	mu    sync.Mutex
	clk   clock.Clock
	delay time.Duration
}

var _ timeout.Mixed = (*InMemory)(nil)

// NewInMemory returns a subject that answers immediately on the real clock.
func NewInMemory() *InMemory { return &InMemory{clk: clock.RealClock()} }

// WithDelay returns a subject that consumes d on the given clock before
// answering, so a test can drive the budget check to either verdict without
// spending the budget.
func WithDelay(clk clock.Clock, d time.Duration) *InMemory {
	return &InMemory{clk: clk, delay: d}
}

// Slow consumes its delay on the injected clock rather than sleeping.
//
// The difference is the whole point of the mixin being checkable: time.Sleep
// spends real seconds and answers to nobody, while a clock the caller holds can
// be advanced instantly and observed exactly.
func (s *InMemory) Slow(ctx context.Context, key string) error {
	if err := contextErr(ctx); err != nil {
		return err
	}
	s.mu.Lock()
	clk, d := s.clk, s.delay
	s.mu.Unlock()
	if d <= 0 {
		return nil
	}
	select {
	case <-clk.After(d):
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// contextErr reports a cancelled or expired context, and tolerates a nil one.
//
// Nil is not a legal context and reaches production anyway, through a caller
// that forgot one. Returning an error is a failed request; dereferencing it is
// an outage, which is why the generated check asks only that this does not
// panic.
func contextErr(ctx context.Context) error {
	if ctx == nil {
		return errors.New("timeouttest: nil context")
	}
	return ctx.Err()
}
