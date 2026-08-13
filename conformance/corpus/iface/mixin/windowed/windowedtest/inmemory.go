// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package windowedtest holds the generated harness and double for
// [go.thesmos.sh/testkit/conformance/corpus/iface/mixin/windowed], and the
// in-memory subject they are run against — scaffolding for the run, so it lives
// beside the harness rather than in the package stating the shape.
package windowedtest

import (
	"context"
	"errors"
	"sync"
	"time"

	"go.thesmos.sh/testkit/clock"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/windowed"
)

// window is the interval the directive declares. Spelled here too because the
// subject has to honour the number the declaration states.
const window = time.Minute

// InMemory is the implementation the generated conformance harness is run
// against.
type InMemory struct {
	mu   sync.Mutex
	clk  clock.Clock
	seen map[string][]time.Time
}

var _ windowed.Mixed = (*InMemory)(nil)

// NewInMemoryOn returns a counter reading the supplied clock — the door the
// generated ModelClocked option opens.
func NewInMemoryOn(clk clock.Clock) *InMemory {
	s := NewInMemory()
	s.clk = clk
	return s
}

// NewInMemory returns a counter on a clock that only moves when told to.
func NewInMemory() *InMemory {
	return &InMemory{
		clk:  clock.NewTestClock(time.Unix(0, 0).UTC()),
		seen: map[string][]time.Time{},
	}
}

// Record adds one occurrence.
func (s *InMemory) Record(ctx context.Context, key string) error {
	if err := contextErr(ctx); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.seen[key] = append(s.seen[key], s.clk.Now())
	return nil
}

// CountIn reports how many occurrences fall inside the window ending now.
//
// Occurrences outside it are not deleted, only excluded: a counter that
// pruned on read would answer differently depending on when it was asked,
// which is a different property from the one the directive states.
func (s *InMemory) CountIn(ctx context.Context, key string) (int, error) {
	if err := contextErr(ctx); err != nil {
		return 0, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	cutoff := s.clk.Now().Add(-window)
	var n int
	for _, at := range s.seen[key] {
		if at.After(cutoff) {
			n++
		}
	}
	return n, nil
}

// contextErr reports a cancelled or expired context, and tolerates a nil one.
func contextErr(ctx context.Context) error {
	if ctx == nil {
		return errors.New("windowedtest: nil context")
	}
	return ctx.Err()
}
