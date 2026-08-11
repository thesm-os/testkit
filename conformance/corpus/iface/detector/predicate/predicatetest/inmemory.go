// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package predicatetest holds the generated harness and double for
// [go.thesmos.sh/testkit/conformance/corpus/iface/detector/predicate], and the
// in-memory subject they are run against.
package predicatetest

import (
	"sync"

	"go.thesmos.sh/testkit/conformance/corpus/iface/detector/predicate"
)

// InMemory is the implementation the generated conformance harness is run
// against.
//
// It lives beside the harness rather than in the package declaring the
// interface, because that is what it is for: a fixture package states a shape,
// and the subject a conformance run holds to it is scaffolding for the run.
type InMemory struct {
	mu    sync.Mutex
	items []string
}

var _ predicate.Predicate = (*InMemory)(nil)

// NewInMemory returns an empty subject.
func NewInMemory() *InMemory { return &InMemory{} }

// IsEmpty answers a question about current state: no arguments, no failure
// mode. The whole signature is the shape.
func (s *InMemory) IsEmpty() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.items) == 0
}
