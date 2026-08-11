// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package lifecycletest holds the generated harness and double for
// [go.thesmos.sh/testkit/conformance/corpus/iface/detector/lifecycle], and the
// in-memory subject they are run against.
package lifecycletest

import (
	"context"
	"errors"
	"sync"

	"go.thesmos.sh/testkit/conformance/corpus/iface/detector/lifecycle"
)

// InMemory is the implementation the generated conformance harness is run
// against.
//
// It lives beside the harness rather than in the package declaring the
// interface, because that is what it is for: a fixture package states a shape,
// and the subject a conformance run holds to it is scaffolding for the run.
type InMemory struct {
	mu     sync.Mutex
	closed bool
}

var _ lifecycle.Lifecycle = (*InMemory)(nil)

// NewInMemory returns an open subject.
func NewInMemory() *InMemory { return &InMemory{} }

// Close is idempotent, which is the lifecycle shape's own law and no part of
// what the signature says. A caller with a deferred close and an explicit one
// relies on it.
func (s *InMemory) Close(ctx context.Context) error {
	if err := contextErr(ctx); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.closed = true
	return nil
}

// Read is the later operation that must report the closed sentinel, which is
// the other half of the shape's law. It is not on the interface: observing
// teardown needs something teardown affects, and Lifecycle declares only the
// teardown.
func (s *InMemory) Read() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return lifecycle.ErrClosed
	}
	return nil
}

// contextErr reports a cancelled or expired context, and tolerates a nil one.
//
// Nil is not a legal context and reaches production anyway, through a caller
// that forgot one. Returning an error is a failed request; dereferencing it is
// an outage, which is why the generated check asks only that this does not
// panic.
func contextErr(ctx context.Context) error {
	if ctx == nil {
		return errors.New("lifecycletest: nil context")
	}
	return ctx.Err()
}
