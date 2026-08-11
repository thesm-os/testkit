// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package leasedidempotentwritertest holds the generated harness and double for
// [go.thesmos.sh/testkit/conformance/corpus/iface/composite/leased-idempotent-writer],
// and the in-memory subject they are run against.
package leasedidempotentwritertest

import (
	"context"
	"errors"
	"sync"

	leasedidempotentwriter "go.thesmos.sh/testkit/conformance/corpus/iface/composite/leased-idempotent-writer"
)

// Registry is the shared ground two holders contend over.
//
// Separate from the holder because a lease is only refused when somebody else
// has it, and a conformance run builds one subject from one factory. So the
// isolation claims — that a release frees one key and not the rest — cannot be
// checks; they need a second holder and live in a package test.
//
// What a single holder *can* be asked is what the composite is actually about:
// whether a repeated acquire needs a repeated release.
type Registry struct {
	mu   sync.Mutex
	held map[string]*InMemory
}

// NewRegistry returns a registry with nothing leased.
func NewRegistry() *Registry { return &Registry{held: map[string]*InMemory{}} }

// Holder returns a new holder contending in this registry.
func (r *Registry) Holder() *InMemory { return &InMemory{registry: r} }

// InMemory is the implementation the generated conformance harness is run
// against.
//
// A lease is held or it is not — there is no depth to it, which is exactly what
// reconciles the two classifications the fixture stacks. `lease` wants one
// release per successful acquire; `idempotent` wants a repeated acquire to
// change nothing. Both hold only if a re-acquire by the standing holder is a
// no-op rather than a second claim.
type InMemory struct{ registry *Registry }

var _ leasedidempotentwriter.LeasedWriter = (*InMemory)(nil)

// NewInMemory returns a lone holder in a registry of its own, which is what a
// conformance run gets.
func NewInMemory() *InMemory { return NewRegistry().Holder() }

// Acquire takes the lease, and does nothing when this holder already has it.
//
// The no-op is what makes the composite consistent at all. Run naively,
// `idempotent` drives Acquire twice and the lease contract's balance check then
// sees one release for two acquires — against an implementation that is
// correct.
func (s *InMemory) Acquire(ctx context.Context, key string) error {
	if err := contextErr(ctx); err != nil {
		return err
	}
	s.registry.mu.Lock()
	defer s.registry.mu.Unlock()
	if holder, taken := s.registry.held[key]; taken && holder != s {
		return leasedidempotentwriter.ErrHeld
	}
	s.registry.held[key] = s
	return nil
}

// Release gives the lease up, and says nothing about not holding it.
//
// One release settles any number of acquires, which follows from the no-op
// above: there is one lease rather than a stack of them. A caller deferring
// Release and returning early on a failed Acquire is ordinary Go, and an error
// here reports a failure to give up something never taken.
func (s *InMemory) Release(ctx context.Context, key string) error {
	if err := contextErr(ctx); err != nil {
		return err
	}
	s.registry.mu.Lock()
	defer s.registry.mu.Unlock()
	if holder, taken := s.registry.held[key]; taken && holder == s {
		delete(s.registry.held, key)
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
		return errors.New("leasedidempotentwritertest: nil context")
	}
	return ctx.Err()
}
