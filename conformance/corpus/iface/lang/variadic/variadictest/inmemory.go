// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package variadictest holds the generated harness and double for
// [go.thesmos.sh/testkit/conformance/corpus/iface/lang/variadic], and the in-memory
// subject they are run against.
package variadictest

import (
	"context"
	"errors"
	"sync"

	"go.thesmos.sh/testkit/conformance/corpus/iface/lang/variadic"
)

// ErrNoKeys is what a lookup with nothing to look up reports.
var ErrNoKeys = errors.New("variadictest: no keys requested")

// InMemory answers a variadic lookup, which is the shape the fixture is about:
// the generated check witnesses exactly one element, because the fixture holds
// one value per parameter and a variadic parameter is still one field.
//
// That is a real limit rather than a defect — one element is what derivation
// can honestly supply — and an author who wants several says so through a check
// of their own, as below.
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
// harness derives no seed and the hit path is unreachable without one.
func (s *InMemory) Put(key, value string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.items[key] = value
}

// Find returns one result per key it holds, in the order asked.
func (s *InMemory) Find(ctx context.Context, keys ...string) ([]string, error) {
	if err := contextErr(ctx); err != nil {
		return nil, err
	}
	if len(keys) == 0 {
		return nil, ErrNoKeys
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]string, 0, len(keys))
	for _, k := range keys {
		if v, ok := s.items[k]; ok {
			out = append(out, v)
		}
	}
	return out, nil
}

// FindWithLimit is Find, truncated.
func (s *InMemory) FindWithLimit(ctx context.Context, limit int, keys ...string) ([]string, error) {
	found, err := s.Find(ctx, keys...)
	if err != nil {
		return nil, err
	}
	if limit >= 0 && len(found) > limit {
		return found[:limit], nil
	}
	return found, nil
}

// contextErr reports a cancelled or expired context and tolerates a nil one.
func contextErr(ctx context.Context) error {
	if ctx == nil {
		return errors.New("variadictest: nil context")
	}
	return ctx.Err()
}

var _ variadic.Finder = (*InMemory)(nil)
