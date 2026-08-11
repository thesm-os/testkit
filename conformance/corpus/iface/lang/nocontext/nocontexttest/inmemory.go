// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package nocontexttest holds the generated harness and double for
// [go.thesmos.sh/testkit/conformance/corpus/iface/lang/nocontext], and the in-memory
// subject they are run against.
package nocontexttest

import (
	"errors"
	"sync"

	"go.thesmos.sh/testkit/conformance/corpus/iface/lang/nocontext"
)

// ErrDivideByZero is what Divide reports rather than panicking, which is the
// only error this interface has to offer.
var ErrDivideByZero = errors.New("nocontexttest: divide by zero")

// InMemory takes no context anywhere, which is what the fixture is about: three
// of the five signature-derived checks are claims about a parameter this
// interface does not have, so each method earns a smoke call and nothing more.
//
// It lives beside the harness rather than in the package declaring the
// interface: a fixture package states a shape, and the subject a conformance
// run holds to it is scaffolding for the run.
type InMemory struct {
	mu    sync.Mutex
	calls int
}

// NewInMemory returns a fresh calculator.
func NewInMemory() *InMemory { return &InMemory{} }

// Add is total: no input pair fails.
func (s *InMemory) Add(a, b int) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.calls++
	return a + b
}

// Divide reports a zero divisor rather than panicking on it.
func (s *InMemory) Divide(a, b int) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.calls++
	if b == 0 {
		return 0, ErrDivideByZero
	}
	return a / b, nil
}

// Reset returns the calculator to its initial state.
func (s *InMemory) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.calls = 0
}

// Calls reports how many operations have run, so a test can observe that Reset
// did something rather than only that it returned.
func (s *InMemory) Calls() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.calls
}

var _ nocontext.Calculator = (*InMemory)(nil)
