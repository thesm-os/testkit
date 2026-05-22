// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package refwindow provides the [RollingCounter] reference for
// the windowed mixin. Counts are tracked per key with timestamped
// increments; observations report the total within the trailing
// window relative to the configured now-function.
package refwindow

import (
	"context"
	"sync"
	"time"
)

// RollingCounter counts events per key within a sliding time
// window. Construct with [NewRollingCounter]. Thread-safe.
type RollingCounter[K comparable] struct {
	mu     sync.Mutex
	window time.Duration
	now    func() time.Time
	events map[K][]time.Time
}

// NewRollingCounter constructs a [RollingCounter] over the given
// window using now() as the clock. now is invoked on every Incr
// and Count call; pass a testkit clock for deterministic runs.
func NewRollingCounter[K comparable](window time.Duration, now func() time.Time) *RollingCounter[K] {
	return &RollingCounter[K]{
		window: window,
		now:    now,
		events: make(map[K][]time.Time),
	}
}

// Incr records one event for k.
func (r *RollingCounter[K]) Incr(_ context.Context, k K) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.events[k] = append(r.events[k], r.now())
	return nil
}

// Count returns the number of events for k whose timestamp falls
// within the trailing window from now(). Expired events are
// removed from the internal log.
func (r *RollingCounter[K]) Count(_ context.Context, k K) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	cutoff := r.now().Add(-r.window)
	events := r.events[k]
	keep := events[:0]
	for _, t := range events {
		if t.After(cutoff) {
			keep = append(keep, t)
		}
	}
	r.events[k] = keep
	return len(keep), nil
}
