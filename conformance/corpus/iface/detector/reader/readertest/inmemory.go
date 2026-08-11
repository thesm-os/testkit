// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package readertest holds the generated harness and double for
// [go.thesmos.sh/testkit/conformance/corpus/iface/detector/reader], and the
// in-memory subject they are run against.
package readertest

import (
	"context"
	"errors"
	"sync"

	"go.thesmos.sh/testkit/conformance/corpus/iface/detector/reader"
)

// InMemory is the implementation the generated conformance harness is run
// against.
//
// It lives beside the harness rather than in the package declaring the
// interface, because that is what it is for: a fixture package states a shape,
// and the subject a conformance run holds to it is scaffolding for the run.
type InMemory struct {
	mu     sync.Mutex
	values map[string]reader.Value
}

var _ reader.Reader = (*InMemory)(nil)

// NewInMemory returns an empty store.
func NewInMemory() *InMemory { return &InMemory{values: map[string]reader.Value{}} }

// Get returns the zero value alongside every error it reports, which is the
// property the reader shape's own check is about: a caller who checks the error
// and one who checks the value must not disagree about whether the call
// succeeded.
func (s *InMemory) Get(ctx context.Context, key string) (reader.Value, error) {
	if err := contextErr(ctx); err != nil {
		return reader.Value{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	v, ok := s.values[key]
	if !ok {
		return reader.Value{}, reader.ErrNotFound
	}
	return v, nil
}

// Put stores a value, so a test can seed an interface that declares no writer
// to seed itself through.
func (s *InMemory) Put(v reader.Value) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.values[v.Key] = v
}

// contextErr reports a cancelled or expired context, and tolerates a nil one.
//
// Nil is not a legal context and reaches production anyway, through a caller
// that forgot one. Returning an error is a failed request; dereferencing it is
// an outage, which is why the generated check asks only that this does not
// panic.
func contextErr(ctx context.Context) error {
	if ctx == nil {
		return errors.New("readertest: nil context")
	}
	return ctx.Err()
}
