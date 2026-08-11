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
type InMemory struct {
	mu   sync.Mutex
	held int
}

var _ leakfree.Mixed = (*InMemory)(nil)

// NewInMemory returns a subject holding nothing.
func NewInMemory() *InMemory { return &InMemory{} }

// Acquire takes the resource.
func (s *InMemory) Acquire(ctx context.Context) error {
	if err := contextErr(ctx); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.held++
	return nil
}

// Release gives it back, refusing when nothing is held.
func (s *InMemory) Release(ctx context.Context) error {
	if err := contextErr(ctx); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.held == 0 {
		return ErrNotHeld
	}
	s.held--
	return nil
}

// Outstanding reports how many acquires have not been matched.
func (s *InMemory) Outstanding(ctx context.Context) (int, error) {
	if err := contextErr(ctx); err != nil {
		return 0, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.held, nil
}

// contextErr reports a cancelled or expired context, and tolerates a nil one.
func contextErr(ctx context.Context) error {
	if ctx == nil {
		return errors.New("leakfreetest: nil context")
	}
	return ctx.Err()
}
