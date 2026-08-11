// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package monotonictest holds the generated harness and double for
// [go.thesmos.sh/testkit/conformance/corpus/iface/mixin/monotonic], and the
// in-memory subject they are run against.
package monotonictest

import (
	"context"
	"errors"
	"sync"

	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/monotonic"
)

// InMemory is the implementation the generated conformance harness is run
// against.
type InMemory struct {
	mu      sync.Mutex
	version int64
}

var _ monotonic.Mixed = (*InMemory)(nil)

// NewInMemory returns a subject at version zero.
func NewInMemory() *InMemory { return &InMemory{} }

// Version never decreases across calls, which is the mixin's claim and what
// AUTO-MONOTONIC-NON-DECREASING states for the model tier. One call cannot
// observe it: any single reading is consistent with any sequence.
func (s *InMemory) Version(ctx context.Context) (int64, error) {
	if err := contextErr(ctx); err != nil {
		return 0, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.version, nil
}

// Advance moves the version forward.
func (s *InMemory) Advance(ctx context.Context) error {
	if err := contextErr(ctx); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.version++
	return nil
}

// contextErr reports a cancelled or expired context, and tolerates a nil one.
func contextErr(ctx context.Context) error {
	if ctx == nil {
		return errors.New("monotonictest: nil context")
	}
	return ctx.Err()
}
