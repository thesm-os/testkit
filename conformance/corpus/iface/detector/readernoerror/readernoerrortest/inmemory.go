// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package readernoerrortest holds the generated harness and double for
// [go.thesmos.sh/testkit/conformance/corpus/iface/detector/readernoerror], and
// the in-memory subject they are run against — scaffolding for the run, so it
// lives beside the harness rather than in the package stating the shape.
package readernoerrortest

import (
	"context"
	"sync"

	"go.thesmos.sh/testkit/conformance/corpus/iface/detector/readernoerror"
)

// InMemory is the implementation the generated conformance harness is run
// against.
type InMemory struct {
	mu     sync.Mutex
	values map[string]readernoerror.Value
}

var _ readernoerror.ReaderNoError = (*InMemory)(nil)

// NewInMemory returns an empty store.
func NewInMemory() *InMemory {
	return &InMemory{values: map[string]readernoerror.Value{}}
}

// Lookup carries the miss in the zero value, because there is no error slot to
// carry it in. That is the shape, and it is why nothing generated holds this
// method to anything but not panicking: with no error to return, a cancelled
// context has nowhere to be reported and the zero value is indistinguishable
// from a value that happens to be zero.
func (s *InMemory) Lookup(ctx context.Context, key string) readernoerror.Value {
	if ctx == nil || ctx.Err() != nil {
		return readernoerror.Value{}
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.values[key]
}

// Put stores a value, so a test can seed an interface that declares no writer
// to seed itself through.
func (s *InMemory) Put(v readernoerror.Value) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.values[v.Key] = v
}
