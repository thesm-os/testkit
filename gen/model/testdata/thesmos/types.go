// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package thesmos mirrors simplified versions of thesmos kernel interfaces
// for Phase 1.5 validation. These exercise the shape combinations found in
// production (Lookup, ReaderWithBool, Mutator, PoisonAccessor, Pure) but
// are simplified from the real kernel interfaces:
//
//   - KindRegistry: close match. Real has same Lookup/Register/Kinds methods.
//   - State: simplified. Real has 8 methods including iter.Seq2, compliance
//     metadata accessors, GetByBytes. Mirror has 3 (Get, Len, Has).
//   - Machine: simplified. Real Fold takes FoldRequest (batch of patches);
//     mirror takes single Patch. Real has ChainHash, Registry accessors.
//
// The validation contract: Tier 1 must produce ≤50 LOC consumer code
// per fixture. Shapes that fall to Unknown indicate Phase 1.5 gaps.
// Full thesmos interfaces with iter.Seq2 returns, opaque-interface
// returns, and request-struct mutators are not yet covered.
package thesmos

import (
	"context"
	"iter"
)

// --- KindRegistry ---

// Kind is a named type for patch kinds.
type Kind int

// KindSpec holds metadata for a registered kind.
type KindSpec struct {
	Name    string
	Version int
}

// FoldFunc processes a patch of a given kind.
type FoldFunc func(data []byte) error

//go:generate testkit model -o registrytest/registry_model.gen.go KindRegistry

// KindRegistry mirrors thesmos's patch.KindRegistry.
// Register at boot only; Lookup is hot-path.
type KindRegistry interface {
	// Register adds a kind. Re-registering the same kind returns error.
	Register(spec KindSpec, fold FoldFunc) error

	// Lookup returns the spec and fold func for a kind.
	// Returns false if not registered.
	Lookup(kind Kind) (KindSpec, FoldFunc, bool)

	// Kinds returns all registered kinds.
	Kinds() []Kind
}

// --- State ---

//go:generate testkit model -o statetest/state_model.gen.go State

// StateKey is a typed key for state entries.
type StateKey string

// StateEntry holds a value with metadata.
type StateEntry struct {
	Value  []byte
	TurnID int
	Region string
}

// State mirrors thesmos's state.State.
// All methods are no-context, 0-alloc reads.
type State interface {
	// Get returns the entry for a key, or false if absent.
	Get(key StateKey) (StateEntry, bool)

	// Len returns the number of entries.
	Len() int

	// Has reports whether a key exists.
	Has(key StateKey) bool
}

// --- Scheduler ---

//go:generate testkit model -o schedulertest/scheduler_model.gen.go Scheduler

// VertexID identifies a node in the execution graph.
type VertexID string

// VertexState tracks whether a vertex has been scheduled.
type VertexState int

const (
	VertexPending VertexState = iota
	VertexReady
	VertexComplete
)

// ReadyRequest is the input to Ready.
type ReadyRequest struct {
	// Vertices maps each vertex to its current state.
	Vertices map[VertexID]VertexState
	// Deps maps each vertex to its dependencies.
	Deps map[VertexID][]VertexID
}

// ReadySet is the output of Ready.
type ReadySet struct {
	// Ready lists vertices whose dependencies are all Complete.
	Ready []VertexID
}

// Scheduler mirrors thesmos's kernel scheduler.
// Single method, no context, no error — pure function.
type Scheduler interface {
	// Ready returns the set of vertices whose dependencies are satisfied.
	// Pure: same ReadyRequest → identical ReadySet. No I/O, no state.
	Ready(req ReadyRequest) ReadySet
}

// --- Ledger ---

//go:generate testkit model -o ledgertest/ledger_model.gen.go Ledger

// LedgerEntry is a per-RunID partitioned log entry.
type LedgerEntry struct {
	RunID string
	Seq   int
	Kind  string
	Data  []byte
}

// Ledger is a production-shaped per-RunID partitioned chain.
// Simplified from the real thesmos Ledger — omits FenceToken,
// batch AppendRequest, Head/Count/DeleteRange. Exercises the
// core chain shape: partitioned append + replay + verify + poison.
type Ledger interface {
	//testkit:appends
	Append(ctx context.Context, entry LedgerEntry) error

	//testkit:verifies
	Verify(ctx context.Context) error

	//testkit:replays
	//testkit:partition-by RunID
	Replay(ctx context.Context, runID string) iter.Seq2[LedgerEntry, error]

	// Err is PoisonAccessor — chain integrity also checked via Verify.
	Err() error
}

// --- Machine ---

//go:generate testkit model -o machinetest/machine_model.gen.go Machine

// Patch is a state-mutating command.
type Patch struct {
	Kind Kind
	Data []byte
}

// MachineState is a read-only snapshot.
type MachineState struct {
	Seq        int
	PatchCount int
}

// Machine mirrors thesmos's fold machine.
type Machine interface {
	// Fold applies a patch. No return — failure surfaces via Err().
	//testkit:mutator
	Fold(ctx context.Context, patch Patch)

	// State returns the current snapshot.
	State() MachineState

	// ExpectedSeq returns the next expected sequence number.
	ExpectedSeq() int

	// Err returns the poison error, or nil if healthy.
	Err() error
}
