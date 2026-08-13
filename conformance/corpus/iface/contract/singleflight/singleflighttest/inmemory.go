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

	"go.thesmos.sh/testkit/conformance/corpus/iface/contract/singleflight"
)

// call is one key's computation: the gate everybody waits on, and the answer
// they share once it is down.
type call struct {
	done chan struct{}
	val  string
}

// InMemory is the implementation the generated conformance harness is run
// against.
//
// One computation per key, ever: the first caller runs the compute it brought
// and every other caller — concurrent or later — shares that answer. Memoized
// rather than windowed, because the coalescing claim counts how often compute
// ran, and a subject whose window is the compute's own duration coalesces only
// when the scheduler happens to interleave the callers inside it.
type InMemory struct {
	mu      sync.Mutex
	flights int
	calls   map[string]*call
}

var _ singleflight.Contract = (*InMemory)(nil)

// NewInMemory returns a coalescer with nothing computed.
func NewInMemory() *InMemory { return &InMemory{calls: map[string]*call{}} }

// Run answers the key's computation, running the caller's compute only if it
// is the first to ask.
//
// The leader computes outside the lock, so waiters block on the call's own
// gate rather than on the whole subject. A nil compute is tolerated and
// computes the zero answer: the harness's signature checks probe with nil,
// and a probe is a failed request, not an outage.
func (s *InMemory) Run(ctx context.Context, key string, compute func() string) (string, error) {
	if err := contextErr(ctx); err != nil {
		return "", err
	}

	s.mu.Lock()
	if c, seen := s.calls[key]; seen {
		s.mu.Unlock()
		<-c.done
		return c.val, nil
	}
	c := &call{done: make(chan struct{})}
	s.calls[key] = c
	s.flights++
	s.mu.Unlock()

	if compute != nil {
		c.val = compute()
	}
	close(c.done)
	return c.val, nil
}

// Flights reports how many computations have ever run.
func (s *InMemory) Flights(ctx context.Context) (int, error) {
	if err := contextErr(ctx); err != nil {
		return 0, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.flights, nil
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
