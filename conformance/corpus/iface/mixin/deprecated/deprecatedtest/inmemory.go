// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package deprecatedtest holds the generated harness and double for
// [go.thesmos.sh/testkit/conformance/corpus/iface/mixin/deprecated], and the
// in-memory subject they are run against.
package deprecatedtest

import (
	"context"
	"errors"
	"sync"

	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/deprecated"
)

// ErrNotFound is what either accessor reports for a key nothing holds.
var ErrNotFound = errors.New("deprecatedtest: not found")

// InMemory serves Old and New from one store, which is what deprecation means
// here: the older spelling still works and answers identically until it is
// removed.
type InMemory struct {
	mu    sync.Mutex
	items map[string]string
}

var _ deprecated.Mixed = (*InMemory)(nil)

// NewInMemory returns an empty store.
func NewInMemory() *InMemory { return &InMemory{items: map[string]string{}} }

// Old is the deprecated spelling. It is held to the same contract as New, which
// is the point: a deprecated method keeps its obligations until it is deleted,
// so the mixin generates no check that skips or excuses it.
func (s *InMemory) Old(ctx context.Context, key string) (string, error) {
	return s.New(ctx, key)
}

// New is the replacement.
func (s *InMemory) New(ctx context.Context, key string) (string, error) {
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

// Put seeds the store, which neither accessor can do.
func (s *InMemory) Put(key, value string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.items[key] = value
}

// contextErr reports a cancelled or expired context, and tolerates a nil one.
func contextErr(ctx context.Context) error {
	if ctx == nil {
		return errors.New("deprecatedtest: nil context")
	}
	return ctx.Err()
}
