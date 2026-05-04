// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package thesmos

import (
	"context"
	"sort"
	"sync"
)

// --- KindRegistry: slice-backed reference ---

// SliceRegistry is a behaviorally-equivalent alternative to InMemoryRegistry.
// Uses a slice scan instead of map lookup. Same contract, different internals.
type SliceRegistry struct {
	mu      sync.Mutex
	entries []registryEntry
}

type registryEntry struct {
	kind Kind
	spec KindSpec
	fold FoldFunc
}

func NewSliceRegistry() *SliceRegistry { return &SliceRegistry{} }

func (r *SliceRegistry) Register(spec KindSpec, fold FoldFunc) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	k := Kind(spec.Version)
	for _, e := range r.entries {
		if e.kind == k {
			return ErrAlreadyRegistered
		}
	}
	r.entries = append(r.entries, registryEntry{kind: k, spec: spec, fold: fold})
	return nil
}

func (r *SliceRegistry) Lookup(kind Kind) (KindSpec, FoldFunc, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, e := range r.entries {
		if e.kind == kind {
			return e.spec, e.fold, true
		}
	}
	return KindSpec{}, nil, false
}

func (r *SliceRegistry) Kinds() []Kind {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]Kind, len(r.entries))
	for i, e := range r.entries {
		out[i] = e.kind
	}
	return out
}

// --- State: slice-backed reference ---

// SliceState is a behaviorally-equivalent alternative to InMemoryState.
type SliceState struct {
	mu      sync.Mutex
	entries []stateKV
}

type stateKV struct {
	key   StateKey
	entry StateEntry
}

func NewSliceState() *SliceState { return &SliceState{} }

func (s *SliceState) Put(key StateKey, entry StateEntry) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, kv := range s.entries {
		if kv.key == key {
			s.entries[i].entry = entry
			return
		}
	}
	s.entries = append(s.entries, stateKV{key: key, entry: entry})
}

func (s *SliceState) Get(key StateKey) (StateEntry, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, kv := range s.entries {
		if kv.key == key {
			return kv.entry, true
		}
	}
	return StateEntry{}, false
}

func (s *SliceState) Len() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.entries)
}

func (s *SliceState) Has(key StateKey) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, kv := range s.entries {
		if kv.key == key {
			return true
		}
	}
	return false
}

// --- Scheduler: filter-based reference ---

// FilterScheduler computes Ready by building a complete set then filtering.
// Different algorithm, same result.
type FilterScheduler struct{}

func NewFilterScheduler() *FilterScheduler { return &FilterScheduler{} }

func (s *FilterScheduler) Ready(req ReadyRequest) ReadySet {
	// Build set of complete vertices.
	complete := make(map[VertexID]bool)
	for v, state := range req.Vertices {
		if state == VertexComplete {
			complete[v] = true
		}
	}
	// Filter pending vertices whose deps are all in complete set.
	var ready []VertexID
	for v, state := range req.Vertices {
		if state != VertexPending {
			continue
		}
		depsOK := true
		for _, dep := range req.Deps[v] {
			if !complete[dep] {
				depsOK = false
				break
			}
		}
		if depsOK {
			ready = append(ready, v)
		}
	}
	sort.Slice(ready, func(i, j int) bool { return ready[i] < ready[j] })
	return ReadySet{Ready: ready}
}

// --- Machine: alternative reference ---

// RefMachine is a behaviorally-equivalent alternative to InMemoryMachine.
// Tracks patches in a slice instead of just counting.
type RefMachine struct {
	mu      sync.Mutex
	patches []Patch
	err     error
}

func NewRefMachine() *RefMachine { return &RefMachine{} }

func (m *RefMachine) Fold(_ context.Context, patch Patch) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.err != nil {
		return
	}
	m.patches = append(m.patches, patch)
}

func (m *RefMachine) State() MachineState {
	m.mu.Lock()
	defer m.mu.Unlock()
	return MachineState{Seq: len(m.patches), PatchCount: len(m.patches)}
}

func (m *RefMachine) ExpectedSeq() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.patches) + 1
}

func (m *RefMachine) Err() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.err
}
