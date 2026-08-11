// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package multireadertest holds the generated harness and double for
// [go.thesmos.sh/testkit/conformance/corpus/iface/detector/multireader], and the
// in-memory subject they are run against — scaffolding for the run, so it lives
// beside the harness rather than in the package stating the shape.
package multireadertest

import (
	"context"
	"errors"
	"sync"

	"go.thesmos.sh/testkit/conformance/corpus/iface/detector/multireader"
)

// InMemory is the implementation the generated conformance harness is run
// against.
type InMemory struct {
	mu     sync.Mutex
	values map[string]multireader.Value
	metas  map[string]multireader.Meta
}

var _ multireader.MultiReader = (*InMemory)(nil)

// NewInMemory returns an empty store.
func NewInMemory() *InMemory {
	return &InMemory{
		values: map[string]multireader.Value{},
		metas:  map[string]multireader.Meta{},
	}
}

// GetWithMeta zeroes *every* slot beside its error, which is what separates this
// shape's check from the plain reader's: a subject that zeroed the value and
// leaked the metadata would satisfy a check reading one slot of two.
func (s *InMemory) GetWithMeta(
	ctx context.Context, key string,
) (multireader.Value, multireader.Meta, error) {
	if err := contextErr(ctx); err != nil {
		return multireader.Value{}, multireader.Meta{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	v, ok := s.values[key]
	if !ok {
		return multireader.Value{}, multireader.Meta{}, multireader.ErrNotFound
	}
	return v, s.metas[key], nil
}

// Put stores a value and its metadata, so a test can seed an interface that
// declares no writer to seed itself through.
func (s *InMemory) Put(v multireader.Value, m multireader.Meta) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.values[v.Key] = v
	s.metas[v.Key] = m
}

// contextErr reports a cancelled or expired context, and tolerates a nil one.
//
// Nil is not a legal context and reaches production anyway, through a caller
// that forgot one. Returning an error is a failed request; dereferencing it is
// an outage, which is why the generated check asks only that this does not
// panic.
func contextErr(ctx context.Context) error {
	if ctx == nil {
		return errors.New("multireadertest: nil context")
	}
	return ctx.Err()
}
