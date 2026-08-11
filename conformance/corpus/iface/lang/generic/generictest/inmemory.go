// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package generictest holds the generated harness and double for
// [go.thesmos.sh/testkit/conformance/corpus/iface/lang/generic], and the in-memory
// subject they are run against.
package generictest

import (
	"context"
	"errors"
	"sync"

	"go.thesmos.sh/testkit/conformance/corpus/iface/lang/generic"
)

// ErrNotFound is what Get reports for a key nothing stored.
var ErrNotFound = errors.New("generictest: not found")

// InMemory is generic so one implementation serves whichever instantiation the
// harness is run at, which is the same freedom the generated harness has.
//
// It lives beside the harness rather than in the package declaring the
// interface: a fixture package states a shape, and the subject a conformance
// run holds to it is scaffolding for the run.
type InMemory[K comparable, V any] struct {
	mu    sync.Mutex
	items map[K]V
}

// NewInMemory returns an empty store at the given instantiation.
func NewInMemory[K comparable, V any]() *InMemory[K, V] {
	return &InMemory[K, V]{items: map[K]V{}}
}

// Get reports the zero value alongside every error.
func (s *InMemory[K, V]) Get(ctx context.Context, key K) (V, error) {
	var zero V
	if err := contextErr(ctx); err != nil {
		return zero, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	v, ok := s.items[key]
	if !ok {
		return zero, ErrNotFound
	}
	return v, nil
}

// Put stores a value under a key.
func (s *InMemory[K, V]) Put(ctx context.Context, key K, value V) error {
	if err := contextErr(ctx); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.items[key] = value
	return nil
}

// contextErr reports a cancelled or expired context and tolerates a nil one.
//
// Nil is not a legal context and reaches production anyway, through a caller
// that forgot one. Returning an error is a failed request; dereferencing it is
// an outage, which is why the generated check asks only that this not panic.
func contextErr(ctx context.Context) error {
	if ctx == nil {
		return errors.New("generictest: nil context")
	}
	return ctx.Err()
}

// Compile-time proof at one instantiation, which is all a generic type can
// claim: the constraint is what promises the rest.
var _ generic.Store[string, int] = (*InMemory[string, int])(nil)
