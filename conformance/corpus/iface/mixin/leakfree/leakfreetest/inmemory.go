// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package leakfreetest holds the generated harness and double for
// [go.thesmos.sh/testkit/conformance/corpus/iface/mixin/leakfree], and the
// in-memory subject they are run against — scaffolding for the run, so it lives
// beside the harness rather than in the package stating the shape.
package leakfreetest

import (
	"context"
	"errors"
	"sync"

	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/leakfree"
)

// ErrNotHeld is what Release reports when nothing is outstanding.
//
// Refusing is what makes the balance observable: a release that silently
// succeeded against an empty pool would let the count go negative and hide
// the very asymmetry the mixin is about.
var ErrNotHeld = errors.New("leakfreetest: nothing to release")

// InMemory is the implementation the generated conformance harness is run
// against.
//
// Each acquire parks a goroutine until its release, so the resource the
// fixture counts is the resource the leak-free law counts — a subject that
// never releases accumulates parked goroutines, and the law's census sees
// exactly the leak the claim is about.
type InMemory struct {
	mu   sync.Mutex
	held []*hold
}

// hold is one outstanding acquire: the gate its goroutine parks on, and the
// acknowledgement that it exited — awaited by Release, so the goroutine
// census the law reads is deterministic rather than a race with the
// scheduler's teardown.
type hold struct {
	release chan struct{}
	exited  chan struct{}
}

var _ leakfree.Mixed = (*InMemory)(nil)

// NewInMemory returns a subject holding nothing.
func NewInMemory() *InMemory { return &InMemory{} }

// Acquire takes the resource, parking a goroutine that lives exactly as
// long as the hold does.
func (s *InMemory) Acquire(ctx context.Context) error {
	if err := contextErr(ctx); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	h := &hold{release: make(chan struct{}), exited: make(chan struct{})}
	go func() {
		<-h.release
		close(h.exited)
	}()
	s.held = append(s.held, h)
	return nil
}

// Release gives it back, refusing when nothing is held.
func (s *InMemory) Release(ctx context.Context) error {
	if err := contextErr(ctx); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.held) == 0 {
		return ErrNotHeld
	}
	last := len(s.held) - 1
	h := s.held[last]
	s.held = s.held[:last]
	close(h.release)
	<-h.exited
	return nil
}

// Outstanding reports how many acquires have not been matched.
func (s *InMemory) Outstanding(ctx context.Context) (int, error) {
	if err := contextErr(ctx); err != nil {
		return 0, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.held), nil
}

// contextErr reports a cancelled or expired context, and tolerates a nil one.
func contextErr(ctx context.Context) error {
	if ctx == nil {
		return errors.New("leakfreetest: nil context")
	}
	return ctx.Err()
}
