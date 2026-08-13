// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package orderaftertest holds the generated harness and double for
// [go.thesmos.sh/testkit/conformance/corpus/iface/mixin/orderafter], and the
// in-memory subject they are run against — scaffolding for the run, so it lives
// beside the harness rather than in the package stating the shape.
package orderaftertest

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/orderafter"
)

// ErrNotPrepared is what Commit refuses an unprepared subject with.
//
// The whole content of the mixin. An ordering constraint nothing enforces is a
// comment: the second caller to get it wrong finds out in production, and the
// first one wrote the code that looks correct.
// Wrapping the declaration's own unready sentinel: the sharpened check
// asserts errors.Is against orderafter.ErrNotReady, and a subject spelling
// its refusal through the chain is the shape consumers ship.
var ErrNotPrepared = fmt.Errorf("orderaftertest: commit before prepare: %w", orderafter.ErrNotReady)

// InMemory is the implementation the generated conformance harness is run
// against.
type InMemory struct {
	mu       sync.Mutex
	prepared bool
	commits  int
}

var _ orderafter.Mixed = (*InMemory)(nil)

// NewInMemory returns an unprepared subject.
func NewInMemory() *InMemory { return &InMemory{} }

// Prepare is the predecessor the mixin's fn parameter names.
func (s *InMemory) Prepare(ctx context.Context) error {
	if err := contextErr(ctx); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.prepared = true
	return nil
}

// Commit is valid only after Prepare, and says so rather than silently doing
// nothing — a refusal the caller can act on beats a no-op they cannot see.
func (s *InMemory) Commit(ctx context.Context) error {
	if err := contextErr(ctx); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.prepared {
		return ErrNotPrepared
	}
	s.commits++
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
		return errors.New("orderaftertest: nil context")
	}
	return ctx.Err()
}
