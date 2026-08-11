// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package genericboundtest holds the generated harness and double for
// [go.thesmos.sh/testkit/conformance/corpus/iface/lang/genericbound], and the in-memory
// subject they are run against.
package genericboundtest

import (
	"context"
	"errors"
	"sync"

	"go.thesmos.sh/testkit/conformance/corpus/iface/lang/genericbound"
)

// ErrNotFound is what Rank reports for a key nothing holds.
var ErrNotFound = errors.New("genericboundtest: not found")

// InMemory is bounded by the same constraint the interface is, so it can be
// instantiated wherever the interface can and nowhere else.
//
// It lives beside the harness rather than in the package declaring the
// interface: a fixture package states a shape, and the subject a conformance
// run holds to it is scaffolding for the run.
type InMemory[K genericbound.Ordered, V any] struct {
	mu    sync.Mutex
	ranks map[K]V
}

// NewInMemory returns an empty ranking at the given instantiation.
func NewInMemory[K genericbound.Ordered, V any]() *InMemory[K, V] {
	return &InMemory[K, V]{ranks: map[K]V{}}
}

// Rank reports the zero value alongside every error.
func (s *InMemory[K, V]) Rank(ctx context.Context, key K) (V, error) {
	var zero V
	if err := contextErr(ctx); err != nil {
		return zero, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	v, ok := s.ranks[key]
	if !ok {
		return zero, ErrNotFound
	}
	return v, nil
}

// Reset empties the ranking.
func (s *InMemory[K, V]) Reset(ctx context.Context) error {
	if err := contextErr(ctx); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	clear(s.ranks)
	return nil
}

// Set is not part of the interface. It exists so a test can reach Rank's hit
// path, which no generated check does: the interface declares no writer, so the
// harness derives no seed.
func (s *InMemory[K, V]) Set(key K, value V) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ranks[key] = value
}

// contextErr reports a cancelled or expired context and tolerates a nil one.
func contextErr(ctx context.Context) error {
	if ctx == nil {
		return errors.New("genericboundtest: nil context")
	}
	return ctx.Err()
}

// Compile-time proof at one instantiation the constraint admits.
var _ genericbound.Ranked[string, int] = (*InMemory[string, int])(nil)
