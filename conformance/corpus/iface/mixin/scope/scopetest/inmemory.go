// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package scopetest holds the generated harness and double for
// [go.thesmos.sh/testkit/conformance/corpus/iface/mixin/scope], and the
// in-memory subject they are run against.
package scopetest

import (
	"context"
	"errors"
	"sync"

	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/scope"
)

// ErrNotFound is what Get reports for a key the scope does not hold.
var ErrNotFound = errors.New("scopetest: not found")

// InMemory keys by scope first, which is the shape: a store hashing the scope
// and key together is indistinguishable here and wrong the moment a tenant has
// to be listed or dropped.
type InMemory struct {
	mu     sync.Mutex
	scopes map[string]map[string]string
}

var _ scope.Mixed = (*InMemory)(nil)

// NewInMemory returns an empty store.
func NewInMemory() *InMemory {
	return &InMemory{scopes: map[string]map[string]string{}}
}

// Set writes within a scope.
func (s *InMemory) Set(ctx context.Context, sc, key, value string) error {
	if err := contextErr(ctx); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.scopes[sc] == nil {
		s.scopes[sc] = map[string]string{}
	}
	s.scopes[sc][key] = value
	return nil
}

// Get reads within a scope and never across one, which is the mixin's claim.
func (s *InMemory) Get(ctx context.Context, sc, key string) (string, error) {
	if err := contextErr(ctx); err != nil {
		return "", err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	v, ok := s.scopes[sc][key]
	if !ok {
		return "", ErrNotFound
	}
	return v, nil
}

// contextErr reports a cancelled or expired context, and tolerates a nil one.
func contextErr(ctx context.Context) error {
	if ctx == nil {
		return errors.New("scopetest: nil context")
	}
	return ctx.Err()
}
