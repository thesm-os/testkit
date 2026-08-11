// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package namedreturnstest holds the generated harness and double for
// [go.thesmos.sh/testkit/conformance/corpus/iface/lang/namedreturns], and the in-memory
// subject they are run against.
package namedreturnstest

import (
	"context"
	"errors"
	"sync"

	"go.thesmos.sh/testkit/conformance/corpus/iface/lang/namedreturns"
)

// ErrNotFound is what every reader here reports for an identifier nothing holds.
var ErrNotFound = errors.New("namedreturnstest: not found")

// InMemory answers the three spellings of one signature identically, which is
// the point of the fixture: whether the source named its results changes the
// declaration and nothing else.
//
// It lives beside the harness rather than in the package declaring the
// interface: a fixture package states a shape, and the subject a conformance
// run holds to it is scaffolding for the run.
type InMemory struct {
	mu    sync.Mutex
	items map[string]string
}

// NewInMemory returns an empty store.
func NewInMemory() *InMemory { return &InMemory{items: map[string]string{}} }

// Put is not part of the interface. The interface declares no writer, so the
// harness derives no seed and a reader's hit path is unreachable without one.
func (s *InMemory) Put(id, value string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.items[id] = value
}

// Named uses the source's declared result names.
func (s *InMemory) Named(ctx context.Context, id string) (item string, err error) {
	return s.lookup(ctx, id)
}

// Unnamed is the anonymous form of the same signature.
func (s *InMemory) Unnamed(ctx context.Context, id string) (string, error) {
	return s.lookup(ctx, id)
}

// PartiallyNamed names one result and blanks the other.
func (s *InMemory) PartiallyNamed(ctx context.Context, id string) (item string, _ error) {
	return s.lookup(ctx, id)
}

// lookup is the one behaviour the three spellings share, written once so the
// fixture varies the declaration rather than the semantics.
func (s *InMemory) lookup(ctx context.Context, id string) (string, error) {
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

// contextErr reports a cancelled or expired context and tolerates a nil one.
//
// Nil is not a legal context and reaches production anyway, through a caller
// that forgot one. Returning an error is a failed request; dereferencing it is
// an outage, which is why the generated check asks only that this not panic.
func contextErr(ctx context.Context) error {
	if ctx == nil {
		return errors.New("namedreturnstest: nil context")
	}
	return ctx.Err()
}

var _ namedreturns.Service = (*InMemory)(nil)
