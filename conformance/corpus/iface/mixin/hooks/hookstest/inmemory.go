// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package hookstest holds the generated harness and double for
// [go.thesmos.sh/testkit/conformance/corpus/iface/mixin/hooks], and the
// in-memory subject they are run against — scaffolding for the run, so it lives
// beside the harness rather than in the package stating the shape.
package hookstest

import (
	"context"
	"errors"
	"sync"

	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/hooks"
)

// InMemory is the implementation the generated conformance harness is run
// against.
type InMemory struct {
	mu       sync.Mutex
	handlers []func(event string)
}

var _ hooks.Mixed = (*InMemory)(nil)

// NewInMemory returns a subject with nothing registered.
func NewInMemory() *InMemory { return &InMemory{} }

// OnEvent registers a handler. It reports nothing, which is why the mixin is
// necessary: accepting a callback and never calling it is indistinguishable
// from accepting one and calling it, until something fires.
func (s *InMemory) OnEvent(fn func(event string)) {
	if fn == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.handlers = append(s.handlers, fn)
}

// Fire invokes every registered handler.
//
// The snapshot is taken under the lock and the handlers run outside it: a
// handler that registered another would deadlock otherwise, and a hook calling
// back into the thing that fired it is ordinary rather than exotic.
func (s *InMemory) Fire(ctx context.Context, event string) error {
	if err := contextErr(ctx); err != nil {
		return err
	}
	s.mu.Lock()
	snapshot := make([]func(event string), len(s.handlers))
	copy(snapshot, s.handlers)
	s.mu.Unlock()

	for _, fn := range snapshot {
		fn(event)
	}
	return nil
}

// Registered reports how many handlers are attached, which the interface
// exposes no way to observe.
func (s *InMemory) Registered() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.handlers)
}

// contextErr reports a cancelled or expired context, and tolerates a nil one.
//
// Nil is not a legal context and reaches production anyway, through a caller
// that forgot one. Returning an error is a failed request; dereferencing it is
// an outage, which is why the generated check asks only that this does not
// panic.
func contextErr(ctx context.Context) error {
	if ctx == nil {
		return errors.New("hookstest: nil context")
	}
	return ctx.Err()
}
