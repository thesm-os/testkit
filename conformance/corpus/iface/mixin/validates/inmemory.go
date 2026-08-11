// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package validates

import (
	"context"
	"errors"
	"sync"
)

// ErrNotFound is what Read reports for a key nothing stored.
var ErrNotFound = errors.New("validates: not found")

// ErrInvalid is what Validate refuses a payload with.
var ErrInvalid = errors.New("validates: invalid payload")

// InMemory is the implementation the generated conformance harness is run
// against.
//
// A fixture package declaring only an interface can be generated for and
// compiled, but nothing can be *run* against it — and a harness nobody runs is
// indistinguishable from one whose checks cannot fail. This is the subject that
// makes the corpus prove the difference.
//
// It is written to satisfy every check the harness derives, which is the point:
// the list below is the contract a conformance suite states, and an
// implementation that skips one of them is one the suite is supposed to reject.
type InMemory struct {
	mu    sync.Mutex
	items map[string]Payload
}

// NewInMemory returns an empty store.
func NewInMemory() *InMemory { return &InMemory{items: map[string]Payload{}} }

// Store refuses what Validate refuses, and reports a context that is done.
func (s *InMemory) Store(ctx context.Context, v Payload) error {
	if err := contextErr(ctx); err != nil {
		return err
	}
	if err := s.Validate(v); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.items[v.Key] = v
	return nil
}

// Validate refuses a payload with no key.
func (*InMemory) Validate(v Payload) error {
	if v.Key == "" {
		return ErrInvalid
	}
	return nil
}

// Read returns the zero value alongside every error it reports, which is the
// property the reader's own check is about: a caller who checks the error and
// one who checks the value must not disagree about whether the call succeeded.
func (s *InMemory) Read(ctx context.Context, key string) (Payload, error) {
	if err := contextErr(ctx); err != nil {
		return Payload{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	v, ok := s.items[key]
	if !ok {
		return Payload{}, ErrNotFound
	}
	return v, nil
}

// contextErr reports a cancelled or expired context, and tolerates a nil one.
//
// Nil is not a legal context and reaches production anyway, through a caller
// that forgot one. Returning an error is a failed request; dereferencing it is
// an outage, which is why the generated check asks only that this does not
// panic.
func contextErr(ctx context.Context) error {
	if ctx == nil {
		return errors.New("validates: nil context")
	}
	return ctx.Err()
}
