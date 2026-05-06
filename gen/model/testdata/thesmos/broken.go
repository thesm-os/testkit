// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package thesmos

import (
	"context"
	"sort"
	"sync"
	"sync/atomic"
)

// --- Broken KindRegistry: Lookup returns wrong spec ---

type BrokenLookupRegistry struct {
	InMemoryRegistry
}

func NewBrokenLookupRegistry() *BrokenLookupRegistry {
	return &BrokenLookupRegistry{
		InMemoryRegistry: *NewInMemoryRegistry(),
	}
}

func (r *BrokenLookupRegistry) Lookup(kind Kind) (KindSpec, FoldFunc, bool) {
	spec, fold, ok := r.InMemoryRegistry.Lookup(kind)
	if ok {
		// BUG: corrupts the spec name
		spec.Name = "WRONG"
	}
	return spec, fold, ok
}

// --- Broken State: Get returns wrong entry ---

type BrokenGetState struct {
	InMemoryState
}

func NewBrokenGetState() *BrokenGetState {
	return &BrokenGetState{InMemoryState: *NewInMemoryState()}
}

func (s *BrokenGetState) Get(key StateKey) (StateEntry, bool) {
	entry, ok := s.InMemoryState.Get(key)
	if ok {
		// BUG: corrupts the TurnID
		entry.TurnID = -999
	}
	return entry, ok
}

// --- Broken Ledger: silently drops every 3rd entry ---

type BrokenLedger struct {
	InMemoryLedger
	count atomic.Int64
}

func NewBrokenLedger() *BrokenLedger {
	return &BrokenLedger{InMemoryLedger: *NewInMemoryLedger()}
}

func (l *BrokenLedger) Append(ctx context.Context, entry LedgerEntry) error {
	n := l.count.Add(1)
	if n%3 == 0 {
		// BUG: silently drops every 3rd entry
		return nil
	}
	return l.InMemoryLedger.Append(ctx, entry)
}

// --- Broken Scheduler: ignores one dependency ---

type BrokenScheduler struct{}

func NewBrokenScheduler() *BrokenScheduler { return &BrokenScheduler{} }

func (s *BrokenScheduler) Ready(req ReadyRequest) ReadySet {
	var ready []VertexID
	for v, state := range req.Vertices {
		if state != VertexPending {
			continue
		}
		// BUG: only checks first dep, ignores the rest
		deps := req.Deps[v]
		if len(deps) > 0 && req.Vertices[deps[0]] != VertexComplete {
			continue
		}
		ready = append(ready, v)
	}
	sort.Slice(ready, func(i, j int) bool { return ready[i] < ready[j] })
	return ReadySet{Ready: ready}
}

// --- Broken Machine: Fold doesn't increment seq ---

type BrokenFoldMachine struct {
	mu    sync.Mutex
	state MachineState
	err   error
}

func NewBrokenFoldMachine() *BrokenFoldMachine {
	return &BrokenFoldMachine{}
}

func (m *BrokenFoldMachine) Fold(_ context.Context, _ Patch) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.err != nil {
		return
	}
	// BUG: increments PatchCount but not Seq
	m.state.PatchCount++
}

func (m *BrokenFoldMachine) State() MachineState {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.state
}

func (m *BrokenFoldMachine) ExpectedSeq() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.state.Seq + 1
}

func (m *BrokenFoldMachine) Err() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.err
}
