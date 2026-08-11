// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package batchedmixinstest holds the generated harness and double for
// [go.thesmos.sh/testkit/conformance/corpus/iface/composite/batched-mixins],
// and the in-memory subject they are run against.
package batchedmixinstest

import (
	"context"
	"errors"
	"maps"
	"slices"
	"sync"

	batchedmixins "go.thesmos.sh/testkit/conformance/corpus/iface/composite/batched-mixins"
)

// listLimit is the cap `//testkit:mixin bounded limit=50` declares on List.
//
// Spelled here as well as in the directive, which is a duplication worth
// stating: the directive is what the annotator reads and this is what the
// subject obeys, and nothing binds them. A generated check for `bounded` would
// close that gap; none exists, so the test names both.
const listLimit = 50

// ErrNotFound reports a key the store does not hold.
var ErrNotFound = errors.New("batchedmixinstest: no value under that key")

// InMemory is the implementation the generated conformance harness is run
// against.
//
// One map, which is what makes Put idempotent: `//testkit:mixin idempotent
// concurrent sideeffect` says a repeated write leaves observable state where
// the first left it, and any branch on presence would be the place that stops
// being true.
type InMemory struct {
	mu     sync.Mutex
	values map[string]string
}

var _ batchedmixins.Batched = (*InMemory)(nil)

// NewInMemory returns an empty store.
func NewInMemory() *InMemory { return &InMemory{values: map[string]string{}} }

// Put writes a value under a key.
func (s *InMemory) Put(ctx context.Context, key, value string) error {
	if err := contextErr(ctx); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.values[key] = value
	return nil
}

// Read returns the zero value alongside every error it reports, so a caller who
// checks the error and one who checks the value do not disagree about whether
// the call succeeded.
//
// `//testkit:mixin readafterwrite write=Put` is the claim that a value Put
// accepted is one this returns, which `AUTO-READ-AFTER-WRITE` states.
func (s *InMemory) Read(ctx context.Context, key string) (string, error) {
	if err := contextErr(ctx); err != nil {
		return "", err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	v, present := s.values[key]
	if !present {
		return "", ErrNotFound
	}
	return v, nil
}

// List returns the keys, sorted and capped at [listLimit].
//
// Sorted because `//testkit:mixin cacheable pure` says the answer depends on
// the state and nothing else — Go's map iteration is deliberately unordered, so
// an unsorted list would differ between two calls that saw the same store.
func (s *InMemory) List(ctx context.Context) ([]string, error) {
	if err := contextErr(ctx); err != nil {
		return nil, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	keys := slices.Sorted(maps.Keys(s.values))
	if len(keys) > listLimit {
		keys = keys[:listLimit]
	}
	return keys, nil
}

// contextErr reports a cancelled or expired context, and tolerates a nil one.
//
// Nil is not a legal context and reaches production anyway, through a caller
// that forgot one. Returning an error is a failed request; dereferencing it is
// an outage, which is why the generated check asks only that this does not
// panic.
func contextErr(ctx context.Context) error {
	if ctx == nil {
		return errors.New("batchedmixinstest: nil context")
	}
	return ctx.Err()
}
