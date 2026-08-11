// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package cacheabletest holds the generated harness and double for
// [go.thesmos.sh/testkit/conformance/corpus/iface/mixin/cacheable], and the
// in-memory subject they are run against.
package cacheabletest

import (
	"context"
	"errors"
	"sync"

	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/cacheable"
)

// ErrNotFound is what Get reports for a key nothing holds.
var ErrNotFound = errors.New("cacheabletest: not found")

// InMemory serves reads from a backing store through a cache, and counts the
// reads that reached the store — which is the only way the mixin's claim is
// observable at all.
type InMemory struct {
	mu     sync.Mutex
	store  map[string]string
	cache  map[string]string
	misses int
}

var _ cacheable.Mixed = (*InMemory)(nil)

// NewInMemory returns an empty store with a cold cache.
func NewInMemory() *InMemory {
	return &InMemory{store: map[string]string{}, cache: map[string]string{}}
}

// Get serves a repeat read from the cache. Whether it did is invisible through
// the interface — the two answers are identical — which is why cacheable is the
// model tier's under ADR-0018 and needs a reference to compare against.
func (s *InMemory) Get(ctx context.Context, key string) (string, error) {
	if err := contextErr(ctx); err != nil {
		return "", err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if v, ok := s.cache[key]; ok {
		return v, nil
	}
	v, ok := s.store[key]
	if !ok {
		return "", ErrNotFound
	}
	s.misses++
	s.cache[key] = v
	return v, nil
}

// Put writes to the backing store, so a test can seed an interface that
// declares no writer to seed itself through.
func (s *InMemory) Put(key, value string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.store[key] = value
}

// contextErr reports a cancelled or expired context, and tolerates a nil one.
func contextErr(ctx context.Context) error {
	if ctx == nil {
		return errors.New("cacheabletest: nil context")
	}
	return ctx.Err()
}
