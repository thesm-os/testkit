// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package poisonabletest holds the generated harness and double for
// [go.thesmos.sh/testkit/conformance/corpus/iface/mixin/poisonable], and the
// in-memory subject they are run against — scaffolding for the run, so it lives
// beside the harness rather than in the package stating the shape.
package poisonabletest

import (
	"context"
	"errors"
	"sync"

	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/poisonable"
)

// ErrPoisoned is the state Probe reports once Fail has run.
var ErrPoisoned = errors.New("poisonabletest: poisoned")

// InMemory is the implementation the generated conformance harness is run
// against.
type InMemory struct {
	mu       sync.Mutex
	poisoned bool
}

var _ poisonable.Mixed = (*InMemory)(nil)

// NewInMemory returns a healthy subject.
func NewInMemory() *InMemory { return &InMemory{} }

// Fail drives the subject into the failed state.
func (s *InMemory) Fail(ctx context.Context) error {
	if err := contextErr(ctx); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.poisoned = true
	return nil
}

// Probe reports the state, and keeps reporting it.
//
// Consistency is the whole claim: a probe that cleared the state on read
// would let one caller see the failure and the next see health, which is the
// intermittency this mixin exists to rule out.
func (s *InMemory) Probe() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.poisoned {
		return ErrPoisoned
	}
	return nil
}

// contextErr reports a cancelled or expired context, and tolerates a nil one.
func contextErr(ctx context.Context) error {
	if ctx == nil {
		return errors.New("poisonabletest: nil context")
	}
	return ctx.Err()
}
