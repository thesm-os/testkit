// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package pooltest holds the generated harness and double for
// [go.thesmos.sh/testkit/conformance/corpus/iface/contract/pool], and the
// in-memory subject they are run against.
package pooltest

import (
	"context"
	"errors"
	"sync"

	"go.thesmos.sh/testkit/conformance/corpus/iface/contract/pool"
)

// ErrExhausted reports a Get with nothing left to hand out.
//
// An error rather than a fresh value, because the pool is what bounds the
// resource: a pool that manufactured one on demand is a constructor, and
// `AUTO-POOL-BALANCED` would hold vacuously against it.
var ErrExhausted = errors.New("pooltest: nothing left in the pool")

// InMemory is the implementation the generated conformance harness is run
// against.
//
// A stack and a count of what is out. The count is the subject rather than
// bookkeeping: `AUTO-POOL-BALANCED` is the claim that every Get is matched by a
// Put, and nothing about a returned value says whether it came from the pool.
type InMemory struct {
	mu    sync.Mutex
	free  []pool.Value
	inUse int
}

var _ pool.Contract = (*InMemory)(nil)

// NewInMemory returns a pool holding the given values.
func NewInMemory(values ...pool.Value) *InMemory { return &InMemory{free: values} }

// Get takes a value out of the pool.
//
// The context is consulted first, so a cancelled caller does not take a value
// they will never return — which is the leak the balance claim is about.
func (s *InMemory) Get(ctx context.Context) (pool.Value, error) {
	if err := contextErr(ctx); err != nil {
		return pool.Value{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.free) == 0 {
		return pool.Value{}, ErrExhausted
	}
	v := s.free[len(s.free)-1]
	s.free = s.free[:len(s.free)-1]
	s.inUse++
	return v, nil
}

// Put returns a value to the pool.
func (s *InMemory) Put(ctx context.Context, v pool.Value) error {
	if err := contextErr(ctx); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.free = append(s.free, v)
	if s.inUse > 0 {
		s.inUse--
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
		return errors.New("pooltest: nil context")
	}
	return ctx.Err()
}
