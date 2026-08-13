// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package indexedtest holds the generated harness and double for
// [go.thesmos.sh/testkit/conformance/corpus/iface/mixin/indexed], and the
// in-memory subject they are run against — scaffolding for the run, so it
// lives beside the harness rather than in the package stating the shape.
package indexedtest

import (
	"context"
	"errors"
	"sync"

	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/indexed"
)

// ErrOutOfRange is what At reports for a position the collection does not
// hold — a miss rather than a panic, because a conformance harness draws
// positions and a subject that panics on one cannot be surveyed at all.
var ErrOutOfRange = errors.New("indexedtest: index out of range")

// InMemory is the implementation the generated conformance harness is run
// against.
type InMemory struct {
	mu     sync.Mutex
	values []indexed.Value
}

var _ indexed.Ranked = (*InMemory)(nil)

// NewInMemory returns an empty collection.
func NewInMemory() *InMemory { return &InMemory{} }

// Add appends an element.
func (s *InMemory) Add(ctx context.Context, v indexed.Value) error {
	if err := contextErr(ctx); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.values = append(s.values, v)
	return nil
}

// Len reports how many elements the positions address.
func (s *InMemory) Len(ctx context.Context) (int, error) {
	if err := contextErr(ctx); err != nil {
		return 0, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.values), nil
}

// At answers the element at a position, and reports a miss otherwise.
func (s *InMemory) At(ctx context.Context, i int) (indexed.Value, error) {
	if err := contextErr(ctx); err != nil {
		return indexed.Value{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if i < 0 || i >= len(s.values) {
		return indexed.Value{}, ErrOutOfRange
	}
	return s.values[i], nil
}

// contextErr reports a cancelled or expired context, and tolerates a nil one.
func contextErr(ctx context.Context) error {
	if ctx == nil {
		return errors.New("indexedtest: nil context")
	}
	return ctx.Err()
}
