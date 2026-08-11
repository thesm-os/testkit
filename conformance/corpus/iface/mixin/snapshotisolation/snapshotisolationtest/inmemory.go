// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package snapshotisolationtest holds the generated harness and double for
// [go.thesmos.sh/testkit/conformance/corpus/iface/mixin/snapshotisolation], and
// the in-memory subject they are run against — scaffolding for the run, so it
// lives beside the harness rather than in the package stating the shape.
package snapshotisolationtest

import (
	"context"
	"errors"
	"slices"
	"sync"

	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/snapshotisolation"
)

// InMemory is the implementation the generated conformance harness is run
// against.
type InMemory struct {
	mu      sync.Mutex
	entries []snapshotisolation.Entry
}

var _ snapshotisolation.Mixed = (*InMemory)(nil)

// NewInMemory returns an empty history.
func NewInMemory() *InMemory { return &InMemory{} }

// Record appends one operation.
func (s *InMemory) Record(ctx context.Context, e snapshotisolation.Entry) error {
	if err := contextErr(ctx); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.entries = append(s.entries, e)
	return nil
}

// History reports the recorded operations.
//
// A copy rather than the slice itself: an anomaly check walks the history
// while the subject may still be recording, and handing out the backing array
// would let a later append be observed mid-walk.
func (s *InMemory) History(ctx context.Context) ([]snapshotisolation.Entry, error) {
	if err := contextErr(ctx); err != nil {
		return nil, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return slices.Clone(s.entries), nil
}

// contextErr reports a cancelled or expired context, and tolerates a nil one.
func contextErr(ctx context.Context) error {
	if ctx == nil {
		return errors.New("snapshotisolationtest: nil context")
	}
	return ctx.Err()
}
