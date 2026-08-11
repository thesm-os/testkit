// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package pointerreadertest holds the generated harness and double for
// [go.thesmos.sh/testkit/conformance/corpus/iface/detector/pointerreader], and
// the in-memory subject they are run against — scaffolding for the run, so it
// lives beside the harness rather than in the package stating the shape.
package pointerreadertest

import (
	"context"
	"sync"

	"go.thesmos.sh/testkit/conformance/corpus/iface/detector/pointerreader"
)

// InMemory is the implementation the generated conformance harness is run
// against.
type InMemory struct {
	mu     sync.Mutex
	values map[string]pointerreader.Value
}

var _ pointerreader.PointerReader = (*InMemory)(nil)

// NewInMemory returns an empty store.
func NewInMemory() *InMemory {
	return &InMemory{values: map[string]pointerreader.Value{}}
}

// Find carries the miss in nil, which is the whole of the shape: adding an
// error return would take the method out of it, because nil would no longer
// have to mean anything.
//
// It returns a copy rather than a pointer into the map, so a caller mutating
// what it got does not rewrite the store. Nothing generated checks that — a
// harness holding one subject cannot see the aliasing — but a shape whose whole
// signal is a pointer is the one where handing out an interior one bites.
func (s *InMemory) Find(ctx context.Context, key string) *pointerreader.Value {
	if ctx == nil || ctx.Err() != nil {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	v, ok := s.values[key]
	if !ok {
		return nil
	}
	return &v
}

// Put stores a value, so a test can seed an interface that declares no writer
// to seed itself through.
func (s *InMemory) Put(v pointerreader.Value) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.values[v.Key] = v
}
