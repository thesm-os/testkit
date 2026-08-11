// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package leaderelectiontest holds the generated harness and double for
// [go.thesmos.sh/testkit/conformance/corpus/iface/contract/leader-election],
// and the in-memory subject they are run against.
package leaderelectiontest

import (
	"context"
	"errors"
	"sync"

	leaderelection "go.thesmos.sh/testkit/conformance/corpus/iface/contract/leader-election"
)

// ErrHeld reports a campaign that lost to a standing leader.
var ErrHeld = errors.New("leaderelectiontest: another candidate holds the leadership")

// Registry is the shared ground the candidates contend over.
//
// Separate from the candidate because leadership is a property of the group
// rather than of any member: "exactly one leader" cannot be stated against a
// single subject, which is why this classification is owned by no tier. Two
// candidates over one registry is the arrangement that makes it stateable, and
// it lives here rather than in the source because a fixture states a shape.
type Registry struct {
	mu     sync.Mutex
	leader *InMemory
}

// NewRegistry returns a registry with no leader.
func NewRegistry() *Registry { return &Registry{} }

// Candidate returns a new candidate contending in this registry.
func (r *Registry) Candidate() *InMemory { return &InMemory{registry: r} }

// InMemory is the implementation the generated conformance harness is run
// against.
type InMemory struct{ registry *Registry }

var _ leaderelection.Contract = (*InMemory)(nil)

// NewInMemory returns a lone candidate in a registry of its own.
//
// A registry per candidate is what a conformance run gets: the harness builds
// one subject from a factory, so contention has nowhere to come from. What it
// still checks is that an uncontested campaign works and reports honestly.
func NewInMemory() *InMemory { return NewRegistry().Candidate() }

// Campaign takes the leadership, or reports that somebody else holds it.
//
// Re-campaigning as the standing leader succeeds. A candidate renewing its own
// claim is the ordinary case, and reporting ErrHeld for it would make every
// leader stand down on its own heartbeat.
func (s *InMemory) Campaign(ctx context.Context) error {
	if err := contextErr(ctx); err != nil {
		return err
	}
	s.registry.mu.Lock()
	defer s.registry.mu.Unlock()
	if s.registry.leader != nil && s.registry.leader != s {
		return ErrHeld
	}
	s.registry.leader = s
	return nil
}

// Resign gives up the leadership, and says nothing about not having it.
//
// A candidate that lost the election still runs its shutdown path, and an error
// there would report a failure to release something it never took.
func (s *InMemory) Resign(ctx context.Context) error {
	if err := contextErr(ctx); err != nil {
		return err
	}
	s.registry.mu.Lock()
	defer s.registry.mu.Unlock()
	if s.registry.leader == s {
		s.registry.leader = nil
	}
	return nil
}

// IsLeader reports whether this candidate currently holds the leadership.
//
// No error to return, so a nil context cannot be reported — only survived. The
// generated check asks exactly that: a caller who forgot a context gets a
// wrong-looking answer rather than an outage.
func (s *InMemory) IsLeader(ctx context.Context) bool {
	if ctx != nil && ctx.Err() != nil {
		return false
	}
	s.registry.mu.Lock()
	defer s.registry.mu.Unlock()
	return s.registry.leader == s
}

// contextErr reports a cancelled or expired context, and tolerates a nil one.
//
// Nil is not a legal context and reaches production anyway, through a caller
// that forgot one. Returning an error is a failed request; dereferencing it is
// an outage, which is why the generated check asks only that this does not
// panic.
func contextErr(ctx context.Context) error {
	if ctx == nil {
		return errors.New("leaderelectiontest: nil context")
	}
	return ctx.Err()
}
