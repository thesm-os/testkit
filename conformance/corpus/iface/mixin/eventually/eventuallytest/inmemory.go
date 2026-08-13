// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package eventuallytest holds the generated harness and double for
// [go.thesmos.sh/testkit/conformance/corpus/iface/mixin/eventually], and the
// in-memory subject they are run against.
package eventuallytest

import (
	"context"
	"errors"
	"maps"
	"slices"
	"sync"

	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/eventually"
)

// InMemory holds published items in a pending queue until Settle drains it,
// which is the mixin: a write is observable eventually rather than immediately,
// and Settle is what a test uses instead of sleeping.
//
// The settled state is a set answered in sorted order, because convergence is
// an equality claim across replicas: two replicas holding one join must spell
// it one way.
type InMemory struct {
	mu      sync.Mutex
	pending []string
	settled map[string]bool
}

var _ eventually.Mixed = (*InMemory)(nil)

// NewInMemory returns an empty subject.
func NewInMemory() *InMemory { return &InMemory{settled: map[string]bool{}} }

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
	for _, item := range s.pending {
		s.settled[item] = true
	}
	s.pending = nil
	return nil
}

// Sync pulls the peer's settled state into this replica, through the same
// interface every replica speaks — the pairwise half of anti-entropy, which
// the generated round composes into full exchange.
func (s *InMemory) Sync(ctx context.Context, peer eventually.Mixed) error {
	if err := contextErr(ctx); err != nil {
		return err
	}
	if peer == nil {
		return errors.New("eventuallytest: nothing to sync with")
	}
	items, err := peer.Items(ctx)
	if err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, item := range items {
		s.settled[item] = true
	}
	return nil
}

// Items reports what has settled, in sorted order.
func (s *InMemory) Items(ctx context.Context) ([]string, error) {
	if err := contextErr(ctx); err != nil {
		return nil, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return slices.Sorted(maps.Keys(s.settled)), nil
}

// contextErr reports a cancelled or expired context, and tolerates a nil one.
func contextErr(ctx context.Context) error {
	if ctx == nil {
		return errors.New("eventuallytest: nil context")
	}
	return ctx.Err()
}
