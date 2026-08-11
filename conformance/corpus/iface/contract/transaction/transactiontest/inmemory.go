// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package transactiontest holds the generated harness and double for
// [go.thesmos.sh/testkit/conformance/corpus/iface/contract/transaction], and
// the in-memory subject they are run against.
package transactiontest

import (
	"context"
	"errors"
	"maps"
	"sync"

	"go.thesmos.sh/testkit/conformance/corpus/iface/contract/transaction"
)

// DoomedKey names a unit of work that always fails partway.
//
// The lever is the key rather than an exported switch, because Run identifies
// the work by it — so "this one goes wrong" is expressible in the interface's
// own terms, and a conformance run reaches the rollback by asking for it.
//
// It is not what the fixture derives, so the harness still seeds through a unit
// that commits. A subject failing every call would fail its seed and never
// reach the check it exists for.
const DoomedKey = "doomed"

// ErrDoomed is what a doomed unit of work reports.
var ErrDoomed = errors.New("transactiontest: the unit of work failed partway")

// InMemory is the implementation the generated conformance harness is run
// against.
//
// Two maps updated as one unit, because that is what makes a rollback visible.
// A transaction over a single assignment has no partial state to roll back to,
// so `AUTO-TRANSACTION-ROLLBACK` would hold against it by having nothing to
// check.
type InMemory struct {
	mu      sync.Mutex
	entries map[string]int
	index   map[string]bool
}

var _ transaction.Contract = (*InMemory)(nil)

// NewInMemory returns a store with nothing committed.
func NewInMemory() *InMemory {
	return &InMemory{entries: map[string]int{}, index: map[string]bool{}}
}

// Run applies a unit of work, or leaves the store exactly as it found it.
//
// The work is staged and committed rather than applied in place. Applying and
// undoing on failure is the other design, and it is the one that leaves state
// behind when the undo is itself interrupted.
func (s *InMemory) Run(ctx context.Context, key string) error {
	if err := contextErr(ctx); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	entries, index := maps.Clone(s.entries), maps.Clone(s.index)
	entries[key]++
	if key == DoomedKey {
		return ErrDoomed
	}
	index[key] = true

	s.entries, s.index = entries, index
	return nil
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
