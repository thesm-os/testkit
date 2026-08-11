// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package mutatortest holds the generated harness and double for
// [go.thesmos.sh/testkit/conformance/corpus/iface/detector/mutator], and the
// in-memory subject they are run against — scaffolding for the run, so it lives
// beside the harness rather than in the package stating the shape.
package mutatortest

import (
	"context"
	"sync"

	"go.thesmos.sh/testkit/conformance/corpus/iface/detector/mutator"
)

// InMemory is the implementation the generated conformance harness is run
// against.
type InMemory struct {
	mu      sync.Mutex
	touched map[string]int
}

var _ mutator.Mutator = (*InMemory)(nil)

// NewInMemory returns an untouched subject.
func NewInMemory() *InMemory { return &InMemory{touched: map[string]int{}} }

// Touch writes and returns nothing, which is the shape and also why the harness
// will not seed through it: a seed that cannot report its own failure leaves
// every check after it asserting against an empty subject.
//
// A cancelled context has nowhere to be reported either, so the only thing left
// is to do no work — which is what the generated family can check, since a
// method that panicked instead would fail the smoke call.
func (s *InMemory) Touch(ctx context.Context, key string) {
	if ctx == nil || ctx.Err() != nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.touched[key]++
}

// Touches reports how often a key was touched, which the interface exposes no
// way to observe — a void method's whole effect is out of band.
func (s *InMemory) Touches(key string) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.touched[key]
}
