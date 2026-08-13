// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package atomictest holds the generated harness and double for
// [go.thesmos.sh/testkit/conformance/corpus/iface/mixin/atomic], and the
// in-memory subject they are run against.
package atomictest

import (
	"context"
	"errors"
	"sync"

	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/atomic"
)

// ErrNotFound is what Read reports for a key nothing holds.
var ErrNotFound = errors.New("atomictest: not found")

// ErrHalfEntry is what Write reports for an entry carrying one half.
//
// A refusal rather than a partial apply, and it is what makes the atomicity
// law reachable: drawn entries carry one-sided empties often, so the failing
// write the law compares around arrives through the interface. An entry with
// *both* halves empty is accepted — that is an honest empty write, not half of
// one.
var ErrHalfEntry = errors.New("atomictest: refuses an entry missing one half")

// InMemory is the implementation the generated conformance harness is run
// against.
type InMemory struct {
	mu    sync.Mutex
	items map[string]atomic.Entry
}

var _ atomic.Mixed = (*InMemory)(nil)

// NewInMemory returns an empty store.
func NewInMemory() *InMemory { return &InMemory{items: map[string]atomic.Entry{}} }

// Write replaces the whole entry under one lock, or refuses it whole: an
// entry with exactly one empty half never lands, and nothing else changes
// when it is refused — which is the mixin's entire claim.
func (s *InMemory) Write(ctx context.Context, e atomic.Entry) error {
	if err := contextErr(ctx); err != nil {
		return err
	}
	if (e.Left == "") != (e.Right == "") {
		return ErrHalfEntry
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.items[e.Key] = e
	return nil
}

// Read returns the whole entry as it was written, or the zero entry beside
// every error it reports.
func (s *InMemory) Read(ctx context.Context, key string) (atomic.Entry, error) {
	if err := contextErr(ctx); err != nil {
		return atomic.Entry{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	e, ok := s.items[key]
	if !ok {
		return atomic.Entry{}, ErrNotFound
	}
	return e, nil
}

// contextErr reports a cancelled or expired context, and tolerates a nil one.
func contextErr(ctx context.Context) error {
	if ctx == nil {
		return errors.New("atomictest: nil context")
	}
	return ctx.Err()
}
