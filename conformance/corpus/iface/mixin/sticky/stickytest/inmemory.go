// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package stickytest holds the generated harness and double for
// [go.thesmos.sh/testkit/conformance/corpus/iface/mixin/sticky], and the
// in-memory subject they are run against — scaffolding for the run, so it lives
// beside the harness rather than in the package stating the shape.
package stickytest

import (
	"context"
	"errors"
	"sync"

	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/sticky"
)

// ErrNotFound is what Get reports for a key nothing wrote.
var ErrNotFound = errors.New("stickytest: not found")

// InMemory is the implementation the generated conformance harness is run
// against.
type InMemory struct {
	mu     sync.Mutex
	values map[string]sticky.Value
}

var _ sticky.Mixed = (*InMemory)(nil)

// NewInMemory returns an empty store.
func NewInMemory() *InMemory {
	return &InMemory{values: map[string]sticky.Value{}}
}

// Store records the value under its own key.
func (s *InMemory) Store(ctx context.Context, v sticky.Value) error {
	if err := contextErr(ctx); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.values[v.Key] = v
	return nil
}

// Get returns what Store recorded, and reports a miss with the zero value —
// which is the default this shape answers an absent key with.
func (s *InMemory) Get(ctx context.Context, key string) (sticky.Value, error) {
	if err := contextErr(ctx); err != nil {
		return sticky.Value{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	v, ok := s.values[key]
	if !ok {
		return sticky.Value{}, ErrNotFound
	}
	return v, nil
}

// contextErr reports a cancelled or expired context, and tolerates a nil one.
func contextErr(ctx context.Context) error {
	if ctx == nil {
		return errors.New("stickytest: nil context")
	}
	return ctx.Err()
}
