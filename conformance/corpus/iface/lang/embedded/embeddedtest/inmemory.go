// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package embeddedtest holds the generated harness and double for
// [go.thesmos.sh/testkit/conformance/corpus/iface/lang/embedded], and the
// in-memory subject they are run against.
package embeddedtest

import (
	"context"
	"errors"
	"sync"

	"go.thesmos.sh/testkit/conformance/corpus/iface/lang/embedded"
)

// ErrNotFound is what Get reports for an identifier nothing holds.
var ErrNotFound = errors.New("embeddedtest: not found")

// InMemory satisfies all three of the package's interfaces at once, which is
// the shape the fixture is about: Composed declares Get and inherits Ping and
// Close, so one implementation is subject to three harnesses and the flattened
// method set is what decides whether the third covers all of them.
//
// It lives beside the harness rather than in the package declaring the
// interfaces, because that is what it is for: a fixture package states a shape,
// and the subject a conformance run holds to it is scaffolding for the run.
type InMemory struct {
	mu     sync.Mutex
	items  map[string]string
	closed bool
}

// Compile-time proof that one implementation answers to all three contracts,
// which is what makes the flattened method set observable at all.
var (
	_ embedded.Base     = (*InMemory)(nil)
	_ embedded.Closer   = (*InMemory)(nil)
	_ embedded.Composed = (*InMemory)(nil)
)

// NewInMemory returns an open, empty store.
func NewInMemory() *InMemory { return &InMemory{items: map[string]string{}} }

// Get reports the zero value alongside every error, which is the property the
// reader's own check is about.
func (s *InMemory) Get(ctx context.Context, id string) (string, error) {
	if err := contextErr(ctx); err != nil {
		return "", err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	v, ok := s.items[id]
	if !ok {
		return "", ErrNotFound
	}
	return v, nil
}

// Put is not part of any interface here. It exists so a test can seed the store
// and reach Get's hit path, which no generated check does: the fixture declares
// no writer, so the harness derives no seed.
func (s *InMemory) Put(id, value string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.items[id] = value
}

// Ping reports a context that is done and succeeds otherwise.
func (*InMemory) Ping(ctx context.Context) error { return contextErr(ctx) }

// Close reports a context that is done, and is idempotent.
func (s *InMemory) Close(ctx context.Context) error {
	if err := contextErr(ctx); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.closed = true
	return nil
}

// Closed reports whether Close has been called, so a test can observe that it
// did something rather than only that it returned nil.
func (s *InMemory) Closed() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.closed
}

// contextErr reports a cancelled or expired context and tolerates a nil one.
//
// Nil is not a legal context and reaches production anyway, through a caller
// that forgot one. Returning an error is a failed request; dereferencing it is
// an outage, which is why the generated check asks only that this not panic.
func contextErr(ctx context.Context) error {
	if ctx == nil {
		return errors.New("embeddedtest: nil context")
	}
	return ctx.Err()
}
