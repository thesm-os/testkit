// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package writesfollowreadstest holds the generated harness and double for
// [go.thesmos.sh/testkit/conformance/corpus/iface/mixin/writesfollowreads], and the
// in-memory subject they are run against — scaffolding for the run, so it lives
// beside the harness rather than in the package stating the shape.
package writesfollowreadstest

import (
	"context"
	"errors"
	"sync"

	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/writesfollowreads"
)

// ErrNotFound is what Get reports for a key nothing wrote.
var ErrNotFound = errors.New("writesfollowreadstest: not found")

// InMemory is the implementation the generated conformance harness is run
// against.
type InMemory struct {
	mu     sync.Mutex
	values map[string]writesfollowreads.Value
}

var _ writesfollowreads.Mixed = (*InMemory)(nil)

// NewInMemory returns an empty store.
func NewInMemory() *InMemory {
	return &InMemory{values: map[string]writesfollowreads.Value{}}
}

// Store records the value under its own key.
func (s *InMemory) Store(ctx context.Context, v writesfollowreads.Value) error {
	if err := contextErr(ctx); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.values[v.Key] = v
	return nil
}

// Get returns what Store recorded, and reports a miss with the zero value.
func (s *InMemory) Get(ctx context.Context, key string) (writesfollowreads.Value, error) {
	if err := contextErr(ctx); err != nil {
		return writesfollowreads.Value{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	v, ok := s.values[key]
	if !ok {
		return writesfollowreads.Value{}, ErrNotFound
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
		return errors.New("writesfollowreadstest: nil context")
	}
	return ctx.Err()
}
