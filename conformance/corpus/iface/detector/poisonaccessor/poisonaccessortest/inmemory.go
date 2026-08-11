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

// Poisoned returns a subject that has already failed.
//
// The latch is what the shape is about, and Err is the only method — so nothing
// a caller does can put a healthy subject into the failed state. A factory can,
// which is what lets the run hold both to the same check.
func Poisoned() *InMemory { return &InMemory{poisoned: true} }

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
