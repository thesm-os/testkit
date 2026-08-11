// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package eventuallytest holds the generated harness and double for
// [go.thesmos.sh/testkit/conformance/corpus/iface/mixin/eventually], and the
// in-memory subject they are run against.
package eventuallytest

import (
	"context"
	"errors"
	"sync"

	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/eventually"
)

// InMemory holds published items in a pending queue until Settle drains it,
// which is the mixin: a write is observable eventually rather than immediately,
// and Settle is what a test uses instead of sleeping.
type InMemory struct {
	mu      sync.Mutex
	pending []string
	settled []string
}

var _ eventually.Mixed = (*InMemory)(nil)

// NewInMemory returns an empty subject.
func NewInMemory() *InMemory { return &InMemory{} }

// Publish queues rather than applying, so Items does not see it yet.
func (s *InMemory) Publish(ctx context.Context, item string) error {
	if err := contextErr(ctx); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.pending = append(s.pending, item)
	return nil
}

// Settle applies everything pending, which is the seam that keeps convergence
// checkable without a clock: a test drives it rather than waiting for it.
func (s *InMemory) Settle(ctx context.Context) error {
	if err := contextErr(ctx); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.settled = append(s.settled, s.pending...)
	s.pending = nil
	return nil
}

// Items reports what has settled.
func (s *InMemory) Items(ctx context.Context) ([]string, error) {
	if err := contextErr(ctx); err != nil {
		return nil, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]string, len(s.settled))
	copy(out, s.settled)
	return out, nil
}

// contextErr reports a cancelled or expired context, and tolerates a nil one.
func contextErr(ctx context.Context) error {
	if ctx == nil {
		return errors.New("eventuallytest: nil context")
	}
	return ctx.Err()
}
