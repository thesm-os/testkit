// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package concurrentreaderstest holds the generated harness and double for
// [go.thesmos.sh/testkit/conformance/corpus/iface/mixin/concurrentreaders], and the in-memory
// subject they are run against.
package concurrentreaderstest

import (
	"context"
	"errors"
	"sync"

	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/concurrentreaders"
)

// ErrNotFound is what a read reports for a key nothing holds.
var ErrNotFound = errors.New("concurrentreaderstest: not found")

// InMemory is the implementation the generated conformance harness is run
// against.
//
// Reads take the lock and hold nothing across it, so concurrent readers never serialise behind one another — observable only under -race, which is why the mixin is the suite's and the check is not generated.
type InMemory struct {
	mu    sync.Mutex
	items map[string]string
}

var _ concurrentreaders.Mixed = (*InMemory)(nil)

// NewInMemory returns an empty store.
func NewInMemory() *InMemory { return &InMemory{items: map[string]string{}} }

// contextErr reports a cancelled or expired context, and tolerates a nil one.
func contextErr(ctx context.Context) error {
	if ctx == nil {
		return errors.New("concurrentreaderstest: nil context")
	}
	return ctx.Err()
}

// Put writes under the lock.
func (s *InMemory) Put(ctx context.Context, key, value string) error {
	if err := contextErr(ctx); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.items[key] = value
	return nil
}

// Get reads under the lock and copies out, so nothing it returns aliases state
// a later writer touches.
func (s *InMemory) Get(ctx context.Context, key string) (string, error) {
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
