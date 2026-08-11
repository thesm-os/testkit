// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package cachetest holds the generated harness and double for
// [go.thesmos.sh/testkit/conformance/corpus/iface/contract/cache], and the
// in-memory subject they are run against.
package cachetest

import (
	"context"
	"errors"
	"sync"

	"go.thesmos.sh/testkit/conformance/corpus/iface/contract/cache"
)

// ErrNotFound reports a key neither the cache nor the backing store holds.
var ErrNotFound = errors.New("cachetest: no value under that key")

// InMemory is the implementation the generated conformance harness is run
// against.
//
// The backing store and the cache in front of it, with a counter between them.
// The counter is the subject rather than instrumentation: "a cached read does
// not reach the backing store" is the contract, and nothing about the return
// value distinguishes a hit from a miss.
type InMemory struct {
	mu      sync.Mutex
	backing map[string]cache.Value
	cached  map[string]cache.Value
	fetches int
}

var _ cache.Contract = (*InMemory)(nil)

// NewInMemory returns a store with an empty backing and an empty cache.
func NewInMemory() *InMemory {
	return &InMemory{
		backing: map[string]cache.Value{},
		cached:  map[string]cache.Value{},
	}
}

// Store puts a value in the backing store without touching the cache.
//
// Not part of the interface: the contract declares a cache and a backing role,
// and neither writes. A conformance run needs the backing populated before a
// read means anything, which is what the harness's seed hook is for.
func (s *InMemory) Store(v cache.Value) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.backing[v.Key] = v
}

// Lookup answers from the cache, and consults the backing store when it cannot.
//
// A miss is not cached. Caching the absence would make the store answer
// "not found" after a later write, which is a different contract — and one
// nothing here declares.
func (s *InMemory) Lookup(ctx context.Context, key string) (cache.Value, error) {
	if err := contextErr(ctx); err != nil {
		return cache.Value{}, err
	}
	s.mu.Lock()
	if v, hit := s.cached[key]; hit {
		s.mu.Unlock()
		return v, nil
	}
	s.mu.Unlock()

	v, err := s.Fetch(ctx, key)
	if err != nil {
		return cache.Value{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cached[key] = v
	return v, nil
}

// Forget drops a key from the backing store without touching the cache.
//
// The seam a cached read is visible through. Both reads return the same value
// whether or not anything was cached, so the only way to tell a cache from a
// pass-through is to take the backing away and ask again — which is a fact
// about the arrangement rather than about the interface, and so belongs to the
// subject a check reaches through.
func (s *InMemory) Forget(key string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.backing, key)
}

// Fetch reads the backing store, and counts the read.
func (s *InMemory) Fetch(ctx context.Context, key string) (cache.Value, error) {
	if err := contextErr(ctx); err != nil {
		return cache.Value{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.fetches++
	v, present := s.backing[key]
	if !present {
		return cache.Value{}, ErrNotFound
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
		return errors.New("cachetest: nil context")
	}
	return ctx.Err()
}
