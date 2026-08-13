// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package idempotentclosetest holds the generated harness and double for
// [go.thesmos.sh/testkit/conformance/corpus/iface/mixin/idempotentclose], and
// the in-memory subject they are run against.
package idempotentclosetest

import (
	"context"
	"errors"
	"sync"

	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/idempotentclose"
)

// InMemory is the implementation the generated conformance harness is run
// against.
//
// One resource, open from construction: enough state for a second Close to
// have something to leave alone, which is what the idempotence law observes
// through Stats.
type InMemory struct {
	mu     sync.Mutex
	closed bool
}

var _ idempotentclose.Closer = (*InMemory)(nil)

// NewInMemory returns a subject holding its one resource open.
func NewInMemory() *InMemory { return &InMemory{} }

// Close releases the resource, and releasing it again changes nothing —
// the flag is idempotent by construction, so the interesting failure a
// broken subject shows here is an error or a counter that keeps moving.
func (s *InMemory) Close(ctx context.Context) error {
	if err := contextErr(ctx); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.closed = true
	return nil
}

// Stats reports how many resources are open.
func (s *InMemory) Stats(ctx context.Context) (int, error) {
	if err := contextErr(ctx); err != nil {
		return 0, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return 0, nil
	}
	return 1, nil
}

// contextErr reports a cancelled or expired context, and tolerates a nil one.
//
// Nil is not a legal context and reaches production anyway, through a caller
// that forgot one. Returning an error is a failed request; dereferencing it is
// an outage, which is why the generated check asks only that this does not
// panic.
func contextErr(ctx context.Context) error {
	if ctx == nil {
		return errors.New("idempotentclosetest: nil context")
	}
	return ctx.Err()
}
