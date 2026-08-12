// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package conservativetest holds the generated harness and double for
// [go.thesmos.sh/testkit/conformance/corpus/iface/mixin/conservative], and the
// in-memory subject they are run against — scaffolding for the run, so it lives
// beside the harness rather than in the package stating the shape.
package conservativetest

import (
	"context"
	"errors"
	"sync"

	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/conservative"
)

// InMemory is the implementation the generated conformance harness is run
// against: a ledger that moves quantity between its reserve and the delta's
// bucket. The transfer is the claim — AUTO-CONSERVATIVE holds the sum
// invariant across Apply, and an Apply that minted quantity out of nothing
// would violate the very property the mixin declares.
type InMemory struct {
	mu      sync.Mutex
	reserve int
	buckets map[string]int
}

var _ conservative.Mixed = (*InMemory)(nil)

// NewInMemory returns a ledger holding nothing anywhere.
func NewInMemory() *InMemory { return &InMemory{buckets: map[string]int{}} }

// Apply moves the delta's amount from the reserve into its bucket — a
// transfer, so the total the law watches never moves.
func (s *InMemory) Apply(ctx context.Context, d conservative.Delta) error {
	if err := contextErr(ctx); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.buckets[d.Key] += d.Amount
	s.reserve -= d.Amount
	return nil
}

// Total reports the conserved quantity: the reserve plus every bucket.
func (s *InMemory) Total(ctx context.Context) (int, error) {
	if err := contextErr(ctx); err != nil {
		return 0, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	total := s.reserve
	for _, held := range s.buckets {
		total += held
	}
	return total, nil
}

// contextErr reports a cancelled or expired context, and tolerates a nil one.
func contextErr(ctx context.Context) error {
	if ctx == nil {
		return errors.New("conservativetest: nil context")
	}
	return ctx.Err()
}
