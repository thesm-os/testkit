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
// against.
type InMemory struct {
	mu    sync.Mutex
	total int
}

var _ conservative.Mixed = (*InMemory)(nil)

// NewInMemory returns a fold at zero.
func NewInMemory() *InMemory { return &InMemory{} }

// Apply folds the delta in by addition, which is the operation the mixin
// declares the property of.
func (s *InMemory) Apply(ctx context.Context, d conservative.Delta) error {
	if err := contextErr(ctx); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.total += d.Amount
	return nil
}

// Total reports the fold so far.
func (s *InMemory) Total(ctx context.Context) (int, error) {
	if err := contextErr(ctx); err != nil {
		return 0, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.total, nil
}

// contextErr reports a cancelled or expired context, and tolerates a nil one.
func contextErr(ctx context.Context) error {
	if ctx == nil {
		return errors.New("conservativetest: nil context")
	}
	return ctx.Err()
}
