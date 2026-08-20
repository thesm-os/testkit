// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package accumulatestest holds the generated harness and double for
// [go.thesmos.sh/testkit/conformance/corpus/iface/mixin/accumulates], and the
// in-memory subject they are run against.
package accumulatestest

import (
	"context"
	"errors"
	"sync"

	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/accumulates"
)

// InMemory is the implementation the generated conformance harness is run
// against.
//
// Add sums rather than replacing, which is the mixin's claim and the one
// difference from idempotent's subject over the same two methods.
type InMemory struct {
	mu     sync.Mutex
	totals map[string]int
}

var _ accumulates.Mixed = (*InMemory)(nil)

// NewInMemory returns an empty store.
func NewInMemory() *InMemory { return &InMemory{totals: map[string]int{}} }

// contextErr reports a cancelled or expired context, and tolerates a nil one.
func contextErr(ctx context.Context) error {
	if ctx == nil {
		return errors.New("accumulatestest: nil context")
	}
	return ctx.Err()
}

// Add compounds: the same key and amount twice leaves twice the amount.
//
// A store that replaced would satisfy every check about one call and violate
// the only one that matters here, which is why the reader is on the interface.
func (s *InMemory) Add(ctx context.Context, key string, amount int) error {
	if err := contextErr(ctx); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.totals[key] += amount
	return nil
}

// Total returns the zero alongside every error it reports.
//
// A key nothing added to totals zero rather than failing: nothing was added is
// an answer, and the sum of no additions is zero.
func (s *InMemory) Total(ctx context.Context, key string) (int, error) {
	if err := contextErr(ctx); err != nil {
		return 0, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.totals[key], nil
}
