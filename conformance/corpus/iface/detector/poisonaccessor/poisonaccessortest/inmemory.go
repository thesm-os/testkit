// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package poisonaccessortest holds the generated harness and double for
// [go.thesmos.sh/testkit/conformance/corpus/iface/detector/poisonaccessor], and
// the in-memory subject they are run against — scaffolding for the run, so it
// lives beside the harness rather than in the package stating the shape.
package poisonaccessortest

import (
	"sync"

	"go.thesmos.sh/testkit/conformance/corpus/iface/detector/poisonaccessor"
)

// InMemory is the implementation the generated conformance harness is run
// against.
type InMemory struct {
	mu        sync.Mutex
	poisoned  bool
	callCount int
}

var _ poisonaccessor.PoisonAccessor = (*InMemory)(nil)

// NewInMemory returns a healthy subject.
func NewInMemory() *InMemory { return &InMemory{} }

// Err reports the latched state and never clears it, which is the shape: the
// failure outlives the call that caused it, so an accessor that reset on read
// would be a queue rather than a latch.
//
// A healthy subject answers nil, which is why the generated smoke check is the
// whole of what a signature can say here — there is no context to cancel and no
// input to make it fail.
func (s *InMemory) Err() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.callCount++
	if s.poisoned {
		return poisonaccessor.ErrPoisoned
	}
	return nil
}

// Poison latches the failure, which nothing on the interface can do: an
// accessor reads state, and something else has to have put it there.
func (s *InMemory) Poison() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.poisoned = true
}

// Calls reports how often Err was asked, so a test can show the latch is read
// rather than computed once.
func (s *InMemory) Calls() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.callCount
}
