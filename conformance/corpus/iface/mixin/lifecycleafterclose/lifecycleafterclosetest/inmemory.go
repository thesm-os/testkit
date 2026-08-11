// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package lifecycleafterclosetest holds the generated harness and double for
// [go.thesmos.sh/testkit/conformance/corpus/iface/mixin/lifecycleafterclose],
// and the in-memory subject they are run against.
package lifecycleafterclosetest

import (
	"context"
	"errors"
	"sync"

	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/lifecycleafterclose"
)

// ErrClosed is what Work reports once the subject has been torn down.
var ErrClosed = errors.New("lifecycleafterclosetest: closed")

// InMemory is the implementation the generated conformance harness is run
// against.
type InMemory struct {
	mu     sync.Mutex
	closed bool
	works  int
}

var _ lifecycleafterclose.Mixed = (*InMemory)(nil)

// NewInMemory returns an open subject.
func NewInMemory() *InMemory { return &InMemory{} }

// Close is idempotent, so a caller with a deferred close and an explicit one
// does not fail the second.
func (s *InMemory) Close(ctx context.Context) error {
	if err := contextErr(ctx); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.closed = true
	return nil
}

// Work refuses once closed, and does no work when it refuses. Reporting the
// error while still doing the work would satisfy any check that only reads the
// error, which is why the count exists.
func (s *InMemory) Work(ctx context.Context) error {
	if err := contextErr(ctx); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return ErrClosed
	}
	s.works++
	return nil
}

// contextErr reports a cancelled or expired context, and tolerates a nil one.
func contextErr(ctx context.Context) error {
	if ctx == nil {
		return errors.New("lifecycleafterclosetest: nil context")
	}
	return ctx.Err()
}
