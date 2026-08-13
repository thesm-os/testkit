// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package leasetest holds the generated harness and double for
// [go.thesmos.sh/testkit/conformance/corpus/iface/contract/lease], and the
// in-memory subject they are run against.
package leasetest

import (
	"context"
	"errors"
	"sync"

	"go.thesmos.sh/testkit/conformance/corpus/iface/contract/lease"
)

// InMemory is the implementation the generated conformance harness is run
// against.
//
// A set of held keys, and nothing else. There is no expiry here because the
// fixture declares none: a lease that timed out would need a clock, and a
// subject inventing one would be testing this package's idea of a deadline
// rather than the contract's.
type InMemory struct {
	mu   sync.Mutex
	held map[string]bool
}

var _ lease.Contract = (*InMemory)(nil)

// NewInMemory returns a store with nothing leased.
func NewInMemory() *InMemory { return &InMemory{held: map[string]bool{}} }

// Acquire takes the lease, or reports that it is held.
//
// Refusing rather than blocking. Both are legitimate lease designs and the
// fixture's signature settles it: a method that blocked would have nothing to
// report through its error, and `AUTO-LEASE-DOUBLE-ACQUIRE-BLOCKS` is the claim
// that the second acquire does not simply succeed.
func (s *InMemory) Acquire(ctx context.Context, key string) error {
	if err := contextErr(ctx); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.held[key] {
		return lease.ErrHeld
	}
	s.held[key] = true
	// The lease is bound to the acquiring context: cancellation releases
	// it, which is the contract's whole claim — a holder that walked away
	// must not pin the key forever. The armed released-on-cancel law is
	// what caught this subject holding keys past their context.
	go func() {
		<-ctx.Done()
		s.mu.Lock()
		defer s.mu.Unlock()
		delete(s.held, key)
	}()
	return nil
}

// Release gives the lease up, and says nothing about not holding it.
//
// A caller deferring Release and returning early on a failed Acquire is
// ordinary Go, and an error there reports a failure to give up something never
// taken.
func (s *InMemory) Release(ctx context.Context, key string) error {
	if err := contextErr(ctx); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.held, key)
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
		return errors.New("leasetest: nil context")
	}
	return ctx.Err()
}
