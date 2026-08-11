// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package permutationtest holds the generated harness and double for
// [go.thesmos.sh/testkit/conformance/corpus/iface/mixin/permutation], and the
// in-memory subject they are run against — scaffolding for the run, so it lives
// beside the harness rather than in the package stating the shape.
package permutationtest

import (
	"context"
	"errors"
	"slices"
	"strings"
	"sync"

	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/permutation"
)

// InMemory is the implementation the generated conformance harness is run
// against.
type InMemory struct {
	mu     sync.Mutex
	values map[string]permutation.Value
}

var _ permutation.Mixed = (*InMemory)(nil)

// NewInMemory returns an empty collection.
func NewInMemory() *InMemory {
	return &InMemory{values: map[string]permutation.Value{}}
}

// Add records the element under its own key, so a repeated append is one
// element rather than two — which is what makes the drain duplicate-free
// by construction rather than by filtering.
func (s *InMemory) Add(ctx context.Context, v permutation.Value) error {
	if err := contextErr(ctx); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.values[v.Key] = v
	return nil
}

// Items drains the collection in key order.
//
// Sorted rather than in map order: Go randomises map iteration, and a drain
// that answered differently on each call would fail a stability claim for a
// reason that has nothing to do with the subject under test.
func (s *InMemory) Items(ctx context.Context) ([]permutation.Value, error) {
	if err := contextErr(ctx); err != nil {
		return nil, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]permutation.Value, 0, len(s.values))
	for _, v := range s.values {
		out = append(out, v)
	}
	slices.SortFunc(out, func(a, b permutation.Value) int {
		return strings.Compare(a.Key, b.Key)
	})
	return out, nil
}

// contextErr reports a cancelled or expired context, and tolerates a nil one.
func contextErr(ctx context.Context) error {
	if ctx == nil {
		return errors.New("permutationtest: nil context")
	}
	return ctx.Err()
}
