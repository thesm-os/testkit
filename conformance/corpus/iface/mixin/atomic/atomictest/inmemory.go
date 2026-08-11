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

// pair is the two halves that move together, stored as one value so a reader
// can never observe one updated and the other not.
type pair struct{ left, right string }

// InMemory is the implementation the generated conformance harness is run
// against.
type InMemory struct {
	mu    sync.Mutex
	items map[string]pair
}

var _ atomic.Mixed = (*InMemory)(nil)

// NewInMemory returns an empty store.
func NewInMemory() *InMemory { return &InMemory{items: map[string]pair{}} }

// Write replaces both halves under one lock, which is the mixin's whole claim:
// two fields written separately are two chances for a reader to see a state
// neither writer intended.
func (s *InMemory) Write(ctx context.Context, key, left, right string) error {
	if err := contextErr(ctx); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.items[key] = pair{left: left, right: right}
	return nil
}

// Read returns both halves as they were written, or the zero for each.
func (s *InMemory) Read(ctx context.Context, key string) (left, right string, err error) {
	if err := contextErr(ctx); err != nil {
		return "", "", err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	p, ok := s.items[key]
	if !ok {
		return "", "", ErrNotFound
	}
	return p.left, p.right, nil
}

// contextErr reports a cancelled or expired context, and tolerates a nil one.
func contextErr(ctx context.Context) error {
	if ctx == nil {
		return errors.New("atomictest: nil context")
	}
	return ctx.Err()
}
