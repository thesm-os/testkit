// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package voidlifecycletest holds the generated harness and double for
// [go.thesmos.sh/testkit/conformance/corpus/iface/detector/voidlifecycle], and
// the in-memory subject they are run against — scaffolding for the run, so it
// lives beside the harness rather than in the package stating the shape.
package voidlifecycletest

import (
	"sync"

	"go.thesmos.sh/testkit/conformance/corpus/iface/detector/voidlifecycle"
)

// InMemory is the implementation the generated conformance harness is run
// against.
type InMemory struct {
	mu    sync.Mutex
	stops int
}

var _ voidlifecycle.VoidLifecycle = (*InMemory)(nil)

// NewInMemory returns a running subject.
func NewInMemory() *InMemory { return &InMemory{} }

// Stop is a teardown that cannot fail, so idempotence is the only law left: with
// no error return there is nothing to report on a second call, and panicking is
// the one way to get it wrong.
func (s *InMemory) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.stops++
}
