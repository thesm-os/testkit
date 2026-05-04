// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package newshapes

import (
	"context"
	"sync"
)

// InMemoryMachine implements [Machine] for model testing.
type InMemoryMachine struct {
	mu    sync.Mutex
	data  map[string]State
	total State
	err   error
}

// NewInMemoryMachine returns a ready-to-use [InMemoryMachine].
func NewInMemoryMachine() *InMemoryMachine {
	return &InMemoryMachine{data: make(map[string]State)}
}

func (m *InMemoryMachine) Fold(_ context.Context, cmd Command) {
	m.mu.Lock()
	defer m.mu.Unlock()
	s := m.data[cmd.ID]
	s.Total += cmd.Value
	s.Count++
	m.data[cmd.ID] = s
	m.total.Total += cmd.Value
	m.total.Count++
}

func (m *InMemoryMachine) Lookup(_ context.Context, id string) (State, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	s, ok := m.data[id]
	return s, ok
}

func (m *InMemoryMachine) State() State {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.total
}

func (m *InMemoryMachine) Err() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.err
}
