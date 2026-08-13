// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package readyourwritestest holds the generated harness and double for
// [go.thesmos.sh/testkit/conformance/corpus/iface/mixin/readyourwrites], and the
// in-memory subject they are run against — scaffolding for the run, so it lives
// beside the harness rather than in the package stating the shape.
package readyourwritestest

import (
	"context"
	"errors"
	"sync"

	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/readyourwrites"
)

// ErrNotFound is what Get reports for a key nothing wrote.
var ErrNotFound = errors.New("readyourwritestest: not found")

// InMemory is the implementation the generated conformance harness is run
// against.
type InMemory struct {
	mu     sync.Mutex
	rev    int64
	values map[string]readyourwrites.Value
}

var _ readyourwrites.Mixed = (*InMemory)(nil)

// NewInMemory returns an empty store.
func NewInMemory() *InMemory {
	return &InMemory{values: map[string]readyourwrites.Value{}}
}

// Store records the value under its own key.
func (s *InMemory) Store(ctx context.Context, v readyourwrites.Value) (readyourwrites.Value, error) {
	if err := contextErr(ctx); err != nil {
		return readyourwrites.Value{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.rev++
	v.Rev = s.rev
	s.values[v.Key] = v
	return v, nil
}

// Get returns what Store recorded, and reports a miss with the zero value.
func (s *InMemory) Get(ctx context.Context, key string) (readyourwrites.Value, error) {
	if err := contextErr(ctx); err != nil {
		return readyourwrites.Value{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	v, ok := s.values[key]
	if !ok {
		return readyourwrites.Value{}, ErrNotFound
	}
	return v, nil
}

// contextErr reports a cancelled or expired context, and tolerates a nil one.
//
// Nil is not a legal context and reaches production anyway, through a caller
// that forgot one. Returning an error is a failed request; dereferencing it is
// an outage.
func contextErr(ctx context.Context) error {
	if ctx == nil {
		return errors.New("readyourwritestest: nil context")
	}
	return ctx.Err()
}
