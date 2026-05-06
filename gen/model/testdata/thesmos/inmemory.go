// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package thesmos

import (
	"context"
	"errors"
	"iter"
	"sort"
	"sync"

	"go.thesmos.sh/testkit/model/refchain"
)

// ErrAlreadyRegistered is returned when registering a duplicate kind.
var ErrAlreadyRegistered = errors.New("already registered")

// --- KindRegistry ---

type InMemoryRegistry struct {
	mu    sync.Mutex
	specs map[Kind]KindSpec
	folds map[Kind]FoldFunc
	order []Kind
}

func NewInMemoryRegistry() *InMemoryRegistry {
	return &InMemoryRegistry{
		specs: make(map[Kind]KindSpec),
		folds: make(map[Kind]FoldFunc),
	}
}

func (r *InMemoryRegistry) Register(spec KindSpec, fold FoldFunc) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	k := Kind(spec.Version) // use Version as Kind for simplicity
	if _, ok := r.specs[k]; ok {
		return ErrAlreadyRegistered
	}
	r.specs[k] = spec
	r.folds[k] = fold
	r.order = append(r.order, k)
	return nil
}

func (r *InMemoryRegistry) Lookup(kind Kind) (KindSpec, FoldFunc, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	spec, ok := r.specs[kind]
	if !ok {
		return KindSpec{}, nil, false
	}
	return spec, r.folds[kind], true
}

func (r *InMemoryRegistry) Kinds() []Kind {
	r.mu.Lock()
	defer r.mu.Unlock()
	cp := make([]Kind, len(r.order))
	copy(cp, r.order)
	return cp
}

// --- State ---

type InMemoryState struct {
	mu   sync.Mutex
	data map[StateKey]StateEntry
}

func NewInMemoryState() *InMemoryState {
	return &InMemoryState{data: make(map[StateKey]StateEntry)}
}

// Put is used by tests to populate state. Not part of the State interface.
func (s *InMemoryState) Put(key StateKey, entry StateEntry) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data[key] = entry
}

func (s *InMemoryState) Get(key StateKey) (StateEntry, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	v, ok := s.data[key]
	return v, ok
}

func (s *InMemoryState) Len() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.data)
}

func (s *InMemoryState) Has(key StateKey) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, ok := s.data[key]
	return ok
}

// --- Ledger ---

// InMemoryLedger implements [Ledger] backed by refchain.PartitionedAppendOnly.
type InMemoryLedger struct {
	chain *refchain.PartitionedAppendOnly[string, LedgerEntry]
}

// NewInMemoryLedger returns a ready-to-use Ledger.
func NewInMemoryLedger() *InMemoryLedger {
	return &InMemoryLedger{
		chain: refchain.NewPartitioned(
			func(e LedgerEntry) string { return e.RunID },
			nil,
		),
	}
}

func (l *InMemoryLedger) Append(ctx context.Context, entry LedgerEntry) error {
	return l.chain.Append(ctx, entry)
}

func (l *InMemoryLedger) Verify(ctx context.Context) error {
	return l.chain.Verify(ctx)
}

func (l *InMemoryLedger) Replay(ctx context.Context, runID string) iter.Seq2[LedgerEntry, error] {
	return l.chain.Replay(ctx, runID)
}

func (l *InMemoryLedger) Err() error {
	return l.chain.Err()
}

// --- Scheduler ---

// MapScheduler computes Ready by iterating vertices and checking deps via map lookups.
type MapScheduler struct{}

func NewMapScheduler() *MapScheduler { return &MapScheduler{} }

func (s *MapScheduler) Ready(req ReadyRequest) ReadySet {
	var ready []VertexID
	for v, state := range req.Vertices {
		if state != VertexPending {
			continue
		}
		allDone := true
		for _, dep := range req.Deps[v] {
			if req.Vertices[dep] != VertexComplete {
				allDone = false
				break
			}
		}
		if allDone {
			ready = append(ready, v)
		}
	}
	sort.Slice(ready, func(i, j int) bool { return ready[i] < ready[j] })
	return ReadySet{Ready: ready}
}

// --- Machine ---

type InMemoryMachine struct {
	mu    sync.Mutex
	state MachineState
	err   error
}

func NewInMemoryMachine() *InMemoryMachine {
	return &InMemoryMachine{}
}

func (m *InMemoryMachine) Fold(_ context.Context, patch Patch) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.err != nil {
		return // poisoned
	}
	m.state.Seq++
	m.state.PatchCount++
}

func (m *InMemoryMachine) State() MachineState {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.state
}

func (m *InMemoryMachine) ExpectedSeq() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.state.Seq + 1
}

func (m *InMemoryMachine) Err() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.err
}
