// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package singleflighttest holds the generated harness and double for
// [go.thesmos.sh/testkit/conformance/corpus/iface/contract/singleflight], and
// the in-memory subject they are run against.
package singleflighttest

import (
	"context"
	"errors"
	"sync"
	"time"

	"go.thesmos.sh/testkit/clock"
	"go.thesmos.sh/testkit/conformance/corpus/iface/contract/singleflight"
)

// SlowKey names work that takes time, and WorkDuration is how long it takes on
// the run's clock.
//
// The coalescing window is exactly that long, and it has to be a window
// somebody can hold open: work completing instantly is work no second caller
// ever finds in flight, so a test of coalescing would pass or fail on how the
// scheduler happened to interleave — which it did, three runs in five, before
// this existed.
//
// Keyed rather than global, for the reason the breaker's unwell downstream is:
// the harness seeds through Run and drives four more checks through it, and a
// subject that parked on every call would hang its own seed with nobody left to
// advance the clock.
const (
	SlowKey      = "slow"
	WorkDuration = time.Second
)

// InMemory is the implementation the generated conformance harness is run
// against.
//
// One in-flight call per key, with everybody else waiting on it. The count of
// underlying calls is the subject rather than instrumentation: every caller
// gets the same answer whether or not anything coalesced, so nothing about the
// return value distinguishes a subject that did the work once from one that did
// it eight times.
type InMemory struct {
	mu       sync.Mutex
	clock    clock.Clock
	inflight map[string]*sync.WaitGroup
	calls    map[string]int
}

var _ singleflight.Contract = (*InMemory)(nil)

// NewInMemory returns a coalescer with nothing in flight.
//
// The clock is a parameter rather than a default, because the coalescing window
// is the leader's work and a window nobody can hold open is one no test can
// observe. Build this with the same [clock.TestClock] the run holds and "a
// second caller found the first in flight" is settled exactly.
func NewInMemory(clk clock.Clock) *InMemory {
	return &InMemory{
		clock:    clk,
		inflight: map[string]*sync.WaitGroup{},
		calls:    map[string]int{},
	}
}

// Run performs the work for a key, or waits for the call already doing it.
//
// The waiter takes no error from the leader. That is the fixture's signature
// rather than a choice — Run reports only its own failure — and a real
// implementation would share the leader's result, which is the part
// `AUTO-SINGLEFLIGHT-COALESCES` compares.
func (s *InMemory) Run(ctx context.Context, key string) error {
	if err := contextErr(ctx); err != nil {
		return err
	}

	s.mu.Lock()
	if wg, running := s.inflight[key]; running {
		s.mu.Unlock()
		wg.Wait()
		return nil
	}
	wg := &sync.WaitGroup{}
	wg.Add(1)
	s.inflight[key] = wg
	s.calls[key]++
	s.mu.Unlock()

	// The work itself. What it does is nothing; how long it takes is the whole
	// point, because that is the window a second caller has to arrive in.
	if key == SlowKey {
		s.clock.Sleep(WorkDuration)
	}

	s.mu.Lock()
	delete(s.inflight, key)
	s.mu.Unlock()
	wg.Done()
	return nil
}

// contextErr reports a cancelled or expired context, and tolerates a nil one.
//
// Nil is not a legal context and reaches production anyway, through a caller
// that forgot one. Returning an error is a failed request; dereferencing it is
// an outage, which is why the generated check asks only that this does not
// panic.
func contextErr(ctx context.Context) error {
	if ctx == nil {
		return errors.New("singleflighttest: nil context")
	}
	return ctx.Err()
}
