// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package receivercollisiontest holds the generated harness and double for
// [go.thesmos.sh/testkit/conformance/corpus/iface/lang/receivercollision], and the in-memory
// subject they are run against.
package receivercollisiontest

import (
	"context"
	"errors"
	"sync"
	"time"

	"go.thesmos.sh/testkit/conformance/corpus/iface/lang/receivercollision"
)

// ErrNotFound is what Get reports for an identifier nothing holds.
var ErrNotFound = errors.New("receivercollisiontest: not found")

// InMemory is the subject for an interface whose methods all name a parameter
// `s` — one at Session, one at string.
//
// That is ordinary Go and it decides the fixture's shape: a field per name *and
// type*, so PutS and GetS are separate values rather than one that would be
// handed to the method taking the other.
//
// It lives beside the harness rather than in the package declaring the
// interface: a fixture package states a shape, and the subject a conformance
// run holds to it is scaffolding for the run.
type InMemory struct {
	mu       sync.Mutex
	sessions map[string]receivercollision.Session
	touched  map[string]time.Time
}

// NewInMemory returns an empty store.
func NewInMemory() *InMemory {
	return &InMemory{
		sessions: map[string]receivercollision.Session{},
		touched:  map[string]time.Time{},
	}
}

// Put stores a session under its own identifier.
func (s *InMemory) Put(ctx context.Context, sess receivercollision.Session) error {
	if err := contextErr(ctx); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sessions[sess.ID] = sess
	return nil
}

// Get reports the zero value alongside every error.
func (s *InMemory) Get(ctx context.Context, id string) (receivercollision.Session, error) {
	if err := contextErr(ctx); err != nil {
		return receivercollision.Session{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	sess, ok := s.sessions[id]
	if !ok {
		return receivercollision.Session{}, ErrNotFound
	}
	return sess, nil
}

// Touch records that a session was seen, and returns nothing — which is why it
// tolerates a context it cannot report on.
func (s *InMemory) Touch(ctx context.Context, sess receivercollision.Session) {
	if contextErr(ctx) != nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.touched[sess.ID] = time.Unix(0, 0)
}

// contextErr reports a cancelled or expired context and tolerates a nil one.
func contextErr(ctx context.Context) error {
	if ctx == nil {
		return errors.New("receivercollisiontest: nil context")
	}
	return ctx.Err()
}

var _ receivercollision.Store = (*InMemory)(nil)
