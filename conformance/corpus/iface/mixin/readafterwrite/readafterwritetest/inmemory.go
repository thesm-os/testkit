// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package readafterwritetest holds the generated harness and double for
// [go.thesmos.sh/testkit/conformance/corpus/iface/mixin/readafterwrite], and the in-memory
// subject they are run against.
package readafterwritetest

import (
	"context"
	"errors"
	"sync"

	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/readafterwrite"
)

// ErrNotFound is what a read reports for a key nothing holds.
var ErrNotFound = errors.New("readafterwritetest: not found")

// InMemory is the implementation the generated conformance harness is run
// against.
//
// A write is visible to the next read, which is the mixin's claim and what AUTO-READ-AFTER-WRITE states for the model tier.
type InMemory struct {
	mu    sync.Mutex
	items map[string]string
}

var _ readafterwrite.Mixed = (*InMemory)(nil)

// NewInMemory returns an empty store.
func NewInMemory() *InMemory { return &InMemory{items: map[string]string{}} }

// contextErr reports a cancelled or expired context, and tolerates a nil one.
func contextErr(ctx context.Context) error {
	if ctx == nil {
		return errors.New("readafterwritetest: nil context")
	}
	return ctx.Err()
}

// Write is the partner the mixin names through `write=Write`.
func (s *InMemory) Write(ctx context.Context, key, value string) error {
	if err := contextErr(ctx); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.items[key] = value
	return nil
}

// Read observes the write immediately: no queue, no eventual settle. An
// implementation buffering writes would satisfy every generated check and fail
// the law the mixin names.
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
