// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package transactiontest holds the generated harness and double for
// [go.thesmos.sh/testkit/conformance/corpus/iface/contract/transaction], and
// the in-memory subject they are run against.
package transactiontest

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"strconv"
	"sync"

	"go.thesmos.sh/testkit/conformance/corpus/iface/contract/transaction"
)

// RunKey is the entry a committed unit of work writes.
//
// One well-known key rather than a caller-chosen one, because Run's signature
// carries no key anymore — the body is the parameter now — and what the
// rollback law probes is that an erroring body left *nothing* behind, wherever
// the subject would have written.
const RunKey = "run"

// InMemory is the implementation the generated conformance harness is run
// against.
//
// The unit of work is staged and committed rather than applied in place.
// Applying and undoing on failure is the other design, and it is the one that
// leaves state behind when the undo is itself interrupted.
//
// Concurrency: runMu serializes whole runs so exactly one staging area is ever
// open, and mu guards the two maps for the duration of a single map operation.
// The two are never held together across the body — a body is caller code and
// caller code calling back into Put or Get would deadlock on a subject holding
// its own map lock.
type InMemory struct {
	runMu sync.Mutex

	mu      sync.Mutex
	runs    int
	entries map[string]transaction.Value
	// staged holds only what the open run wrote, not a copy of the store.
	// Committing key-by-key rather than swapping a clone in is what keeps a
	// concurrent Put outside the run from being silently undone by it.
	staged map[string]transaction.Value
}

var _ transaction.Contract = (*InMemory)(nil)

// NewInMemory returns a store with nothing committed.
func NewInMemory() *InMemory {
	return &InMemory{entries: map[string]transaction.Value{}}
}

// Run stages a unit of work, hands control to the body, and commits only when
// the body succeeds — an erroring body's staging is discarded whole.
//
// The body runs outside the map lock, because a body is caller code and caller
// code that writes through Put or reads back through Get would otherwise
// deadlock on the subject it was handed. A nil body is tolerated as an empty
// unit of work: the harness's signature checks probe with nil, and a probe is
// a failed request, not an outage.
func (s *InMemory) Run(ctx context.Context, body func(ctx context.Context) error) error {
	if err := contextErr(ctx); err != nil {
		return err
	}
	s.runMu.Lock()
	defer s.runMu.Unlock()

	s.mu.Lock()
	runs := s.runs + 1
	s.staged = map[string]transaction.Value{
		RunKey: {Key: RunKey, Body: strconv.Itoa(runs)},
	}
	s.mu.Unlock()

	var bodyErr error
	if body != nil {
		bodyErr = body(ctx)
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	staged := s.staged
	s.staged = nil
	if bodyErr != nil {
		// Discarded whole: every write the body staged goes with it, which is
		// the claim AUTO-TRANSACTION-ROLLBACK exists to state.
		return bodyErr
	}
	maps.Copy(s.entries, staged)
	s.runs = runs
	return nil
}

// Put writes through outside a run and stages inside one.
//
// Which of the two it does is not the caller's to know: a keyed write means
// "this key now holds this value", and a transaction is the scope that decides
// when "now" is. Inside a run the value lands in the staging area, where an
// erroring body discards it and a succeeding one commits it.
func (s *InMemory) Put(ctx context.Context, key string, v transaction.Value) error {
	if err := contextErr(ctx); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.staged != nil {
		s.staged[key] = v
		return nil
	}
	s.entries[key] = v
	return nil
}

// Get returns the zero value alongside every error it reports, so a caller who
// checks the error and one who checks the value do not disagree about whether
// the call succeeded.
func (s *InMemory) Get(ctx context.Context, key string) (transaction.Value, error) {
	if err := contextErr(ctx); err != nil {
		return transaction.Value{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	v, present := s.entries[key]
	if !present {
		return transaction.Value{}, fmt.Errorf("transactiontest: key %q: %w", key, transaction.ErrNotFound)
	}
	return v, nil
}

// contextErr reports a cancelled or expired context, and tolerates a nil one.
//
// Nil is not a legal context and reaches production anyway, through a caller
// that forgot one. Returning an error is a failed request; dereferencing it is
// an outage, which is why the generated check asks only that this does not
// panic.
func contextErr(ctx context.Context) error {
	if ctx == nil {
		return errors.New("transactiontest: nil context")
	}
	return ctx.Err()
}
