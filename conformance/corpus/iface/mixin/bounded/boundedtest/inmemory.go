// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package boundedtest holds the generated harness and double for
// [go.thesmos.sh/testkit/conformance/corpus/iface/mixin/bounded], and the
// in-memory subject they are run against.
package boundedtest

import (
	"context"
	"errors"
	"sync"

	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/bounded"
)

// InMemory is the implementation the generated conformance harness is run
// against.
type InMemory struct {
	mu    sync.Mutex
	limit int
	items []string
}

var _ bounded.Mixed = (*InMemory)(nil)

// NewInMemory returns an empty collection clamped at the given ceiling.
//
// The capacity is a parameter rather than a constant here because the harness
// hands it over: `//testkit:mixin bounded limit=5` is read by the generator and
// passed to every constructor, so a subject that restated the number could
// disagree with the law measuring it.
func NewInMemory(limit int) *InMemory { return &InMemory{limit: limit} }

// Add grows the collection; what List answers stays clamped to the limit
// regardless, which is the mixin's whole claim.
func (s *InMemory) Add(ctx context.Context, item string) error {
	if err := contextErr(ctx); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.items = append(s.items, item)
	return nil
}

// List never returns more than the declared limit, which is the whole of the
// mixin and no part of the signature — `List(ctx) ([]string, error)` is the same shape
// bounded or not.
func (s *InMemory) List(ctx context.Context) ([]string, error) {
	if err := contextErr(ctx); err != nil {
		return nil, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	n := min(len(s.items), s.limit)
	out := make([]string, n)
	copy(out, s.items[:n])
	return out, nil
}

// contextErr reports a cancelled or expired context, and tolerates a nil one.
func contextErr(ctx context.Context) error {
	if ctx == nil {
		return errors.New("boundedtest: nil context")
	}
	return ctx.Err()
}
