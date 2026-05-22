// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package refstate provides the [Compliance] reference for
// compliance-tracking state machines: a per-key state V advanced
// by transitions M, where each (V, M) → V transition is validated
// against a guard function. Invalid transitions error without
// mutating state.
package refstate

import (
	"context"
	"sync"
)

// Compliance maps each key K to a state V advanced by transitions
// M. Construct with [NewCompliance]. Thread-safe.
type Compliance[K comparable, V, M any] struct {
	mu       sync.Mutex
	states   map[K]V
	initial  V
	guard    func(from V, m M) (to V, ok bool)
	rejected error
}

// NewCompliance constructs a [Compliance] machine.
//
// initial is the per-key starting state when no prior transition
// has been applied. guard returns the target state plus true when
// the transition is allowed; false when rejected. rejected is the
// error returned by Apply on a guard-rejected transition.
func NewCompliance[K comparable, V, M any](
	initial V,
	guard func(from V, m M) (to V, ok bool),
	rejected error,
) *Compliance[K, V, M] {
	return &Compliance[K, V, M]{
		states:   make(map[K]V),
		initial:  initial,
		guard:    guard,
		rejected: rejected,
	}
}

// Apply attempts the transition m for key k. On success the state
// advances; on rejection the state is unchanged and the rejected
// error is returned.
func (c *Compliance[K, V, M]) Apply(_ context.Context, k K, m M) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	cur, ok := c.states[k]
	if !ok {
		cur = c.initial
	}
	next, allowed := c.guard(cur, m)
	if !allowed {
		return c.rejected
	}
	c.states[k] = next
	return nil
}

// State returns the current state for k, or the initial state when
// no transition has been applied.
func (c *Compliance[K, V, M]) State(_ context.Context, k K) (V, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	s, ok := c.states[k]
	if !ok {
		return c.initial, nil
	}
	return s, nil
}
