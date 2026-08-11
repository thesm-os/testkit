// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package readerwithbooltest holds the generated harness and double for
// [go.thesmos.sh/testkit/conformance/corpus/iface/detector/readerwithbool], and
// the in-memory subject they are run against — scaffolding for the run, so it
// lives beside the harness rather than in the package stating the shape.
package readerwithbooltest

import (
	"context"
	"sync"

	"go.thesmos.sh/testkit/conformance/corpus/iface/detector/readerwithbool"
)

// InMemory is the implementation the generated conformance harness is run
// against.
type InMemory struct {
	mu     sync.Mutex
	values map[string]readerwithbool.Value
}

var _ readerwithbool.ReaderWithBool = (*InMemory)(nil)

// NewInMemory returns an empty store.
func NewInMemory() *InMemory {
	return &InMemory{values: map[string]readerwithbool.Value{}}
}

// Load reports absence through the flag rather than an error, which is the
// comma-ok idiom the shape is named for. A miss is not a failure here, so a
// cancelled context is reported the same way absence is — the flag is the only
// channel there is.
func (s *InMemory) Load(ctx context.Context, key string) (readerwithbool.Value, bool) {
	if ctx == nil || ctx.Err() != nil {
		return readerwithbool.Value{}, false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	v, ok := s.values[key]
	if !ok {
		return readerwithbool.Value{}, false
	}
	return v, true
}

// Put stores a value, so a test can seed an interface that declares no writer
// to seed itself through.
func (s *InMemory) Put(v readerwithbool.Value) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.values[v.Key] = v
}
