// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package batchreadertest holds the generated harness and double for
// [go.thesmos.sh/testkit/conformance/corpus/iface/detector/batchreader], and the
// in-memory subject they are run against — scaffolding for the run, so it lives
// beside the harness rather than in the package stating the shape.
package batchreadertest

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"go.thesmos.sh/testkit/conformance/corpus/iface/detector/batchreader"
)

// ErrNotFound is what GetAll reports when any requested key is absent.
//
// All-or-nothing rather than a short result, because a caller cannot tell a
// partial answer from a complete one without comparing lengths — and a batch
// read that silently drops keys is the failure mode this shape has.
var ErrNotFound = errors.New("batchreadertest: not found")

// InMemory is the implementation the generated conformance harness is run
// against.
type InMemory struct {
	mu     sync.Mutex
	values map[string]batchreader.Value
}

var _ batchreader.BatchReader = (*InMemory)(nil)

// NewInMemory returns an empty store.
func NewInMemory() *InMemory {
	return &InMemory{values: map[string]batchreader.Value{}}
}

// GetAll answers in the order asked, and returns nil beside every error it
// reports — which is the property the generated zero-value check is about, and
// the one a batch read gets wrong by returning what it did find.
//
// The generated checks pass exactly one key, because a fixture holds one value
// per parameter. Everything about several is in this package's own test.
func (s *InMemory) GetAll(ctx context.Context, keys ...string) ([]batchreader.Value, error) {
	if err := contextErr(ctx); err != nil {
		return nil, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]batchreader.Value, 0, len(keys))
	for _, k := range keys {
		v, ok := s.values[k]
		if !ok {
			return nil, fmt.Errorf("%w: %s", ErrNotFound, k)
		}
		out = append(out, v)
	}
	return out, nil
}

// Put stores a value, so a test can seed an interface that declares no writer
// to seed itself through.
func (s *InMemory) Put(v batchreader.Value) {
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
		return errors.New("batchreadertest: nil context")
	}
	return ctx.Err()
}
