// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// provides the [FoldMachine] reference for
// kernel-grade interfaces that accumulate state via a fold over a
// stream of events. Each Apply(p) advances the state via the
// supplied fold function; State() returns the current accumulation.

package ref

import (
	"context"
	"sync"
)

// FoldMachine is a stateful fold over a stream of P events into an
// accumulated state S. Construct with [NewFoldMachine]. Thread-safe.
type FoldMachine[P, S any] struct {
	mu    sync.Mutex
	state S
	fold  func(S, P) S
}

// NewFoldMachine constructs a [FoldMachine] with the given initial
// state and fold function.
func NewFoldMachine[P, S any](initial S, fold func(S, P) S) *FoldMachine[P, S] {
	return &FoldMachine[P, S]{
		state: initial,
		fold:  fold,
	}
}

// Apply advances the state by folding p into it.
func (m *FoldMachine[P, S]) Apply(_ context.Context, p P) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.state = m.fold(m.state, p)
	return nil
}

// State returns the current accumulated state.
func (m *FoldMachine[P, S]) State(_ context.Context) (S, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.state, nil
}
