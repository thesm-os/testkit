// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package lookuptest holds the generated harness and double for
// [go.thesmos.sh/testkit/conformance/corpus/iface/detector/lookup], and the
// in-memory subject they are run against — scaffolding for the run, so it lives
// beside the harness rather than in the package stating the shape.
package lookuptest

import (
	"sync"

	"go.thesmos.sh/testkit/conformance/corpus/iface/detector/lookup"
)

// InMemory is the implementation the generated conformance harness is run
// against.
type InMemory struct {
	mu     sync.Mutex
	values map[string]lookup.Value
	metas  map[string]lookup.Meta
}

var _ lookup.Lookup = (*InMemory)(nil)

// NewInMemory returns an empty store.
func NewInMemory() *InMemory {
	return &InMemory{
		values: map[string]lookup.Value{},
		metas:  map[string]lookup.Meta{},
	}
}

// Inspect takes no context and cannot fail, so its whole generated contract is
// that it does not panic — a synchronous question about state the subject
// already holds.
//
// Both slots go to the zero on a miss. Nothing generated checks it: with no
// error return there is no zero-value check, and the flag is the only signal.
func (s *InMemory) Inspect(key string) (lookup.Value, lookup.Meta, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	v, ok := s.values[key]
	if !ok {
		return lookup.Value{}, lookup.Meta{}, false
	}
	return v, s.metas[key], true
}

// Put stores a value and its metadata, so a test can seed an interface that
// declares no writer to seed itself through.
func (s *InMemory) Put(v lookup.Value, m lookup.Meta) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.values[v.Key] = v
	s.metas[v.Key] = m
}
