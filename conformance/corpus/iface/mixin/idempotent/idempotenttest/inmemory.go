// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package idempotenttest holds the generated harness and double for
// [go.thesmos.sh/testkit/conformance/corpus/iface/mixin/idempotent], and the in-memory
// subject they are run against.
package idempotenttest

import (
	"context"
	"errors"
	"sync"

	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/idempotent"
)

// ErrNotFound is what a read reports for a key nothing holds.
var ErrNotFound = errors.New("idempotenttest: not found")

// InMemory is the implementation the generated conformance harness is run
// against.
//
// Put replaces rather than accumulating, so repeating one write leaves the store as one write did — the mixin's claim, and invisible through Read alone.
type InMemory struct {
	mu     sync.Mutex
	items  map[string]string
	writes int
}

var _ idempotent.Mixed = (*InMemory)(nil)

// NewInMemory returns an empty store.
func NewInMemory() *InMemory { return &InMemory{items: map[string]string{}} }

// Writes reports how many writes landed, which the interface exposes no way to
// observe.
func (s *InMemory) Writes() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.writes
}

// contextErr reports a cancelled or expired context, and tolerates a nil one.
func contextErr(ctx context.Context) error {
	if ctx == nil {
		return errors.New("idempotenttest: nil context")
	}
	return ctx.Err()
}

// Put is idempotent: the same key and value twice is the same state as once.
// The write count is what makes the difference observable, and it is not on the
// interface — which is why AUTO-IDEMPOTENT-WRITE is the model tier's.
func (s *InMemory) Put(ctx context.Context, key, value string) error {
	if err := contextErr(ctx); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if existing, ok := s.items[key]; ok && existing == value {
		return nil
	}
	s.items[key] = value
	s.writes++
	return nil
}

// Read returns the zero value alongside every error it reports.
func (s *InMemory) Read(ctx context.Context, key string) (string, error) {
	if err := contextErr(ctx); err != nil {
		return "", err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	v, ok := s.items[key]
	if !ok {
		return "", ErrNotFound
	}
	return v, nil
}
