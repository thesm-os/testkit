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
//
// It honours the sticky claim the fixture states: the first value a key
// resolves to is the one every later Get answers, whatever Store recorded in
// between. The model tier proved the claim has teeth — the first value pool
// wide enough to draw a same-key overwrite failed a latest-write-wins version
// of this store against its own law.
type InMemory struct {
	mu       sync.Mutex
	values   map[string]sticky.Value
	resolved map[string]sticky.Value
}

var _ sticky.Mixed = (*InMemory)(nil)

// NewInMemory returns an empty store.
func NewInMemory() *InMemory {
	return &InMemory{
		values:   map[string]sticky.Value{},
		resolved: map[string]sticky.Value{},
	}
}

// Store records the value under its own key. What a resolved key answers is
// Get's business, not this one's: the record is kept either way, and only
// resolution pins.
func (s *InMemory) Store(ctx context.Context, v sticky.Value) error {
	if err := contextErr(ctx); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.values[v.Key] = v
	return nil
}

// Get answers the first value the key ever resolved to, resolving it now
// where it never has. A miss is not a resolution — it reports the sentinel
// with the zero value and leaves the key free to resolve to whatever a later
// Store records.
func (s *InMemory) Get(ctx context.Context, key string) (sticky.Value, error) {
	if err := contextErr(ctx); err != nil {
		return sticky.Value{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if v, pinned := s.resolved[key]; pinned {
		return v, nil
	}
	v, ok := s.values[key]
	if !ok {
		return sticky.Value{}, ErrNotFound
	}
	s.resolved[key] = v
	return v, nil
}

// contextErr reports a cancelled or expired context, and tolerates a nil one.
func contextErr(ctx context.Context) error {
	if ctx == nil {
		return errors.New("stickytest: nil context")
	}
	return ctx.Err()
}
