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
type InMemory struct {
	mu      sync.Mutex
	runs    int
	entries map[string]transaction.Value
}

var _ transaction.Contract = (*InMemory)(nil)

// NewInMemory returns a store with nothing committed.
func NewInMemory() *InMemory {
	return &InMemory{entries: map[string]transaction.Value{}}
}

// Run stages a unit of work, hands control to the body, and commits only when
// the body succeeds — an erroring body's staging is discarded whole.
//
// The body runs outside the lock, because a body is caller code and caller
// code that reads back through Get would otherwise deadlock on the subject it
// was handed. A nil body is tolerated as an empty unit of work: the harness's
// signature checks probe with nil, and a probe is a failed request, not an
// outage.
func (s *InMemory) Run(ctx context.Context, body func(ctx context.Context) error) error {
	if err := contextErr(ctx); err != nil {
		return err
	}

	s.mu.Lock()
	staged := maps.Clone(s.entries)
	runs := s.runs + 1
	staged[RunKey] = transaction.Value{Key: RunKey, Body: strconv.Itoa(runs)}
	s.mu.Unlock()

	if body != nil {
		if err := body(ctx); err != nil {
			return err
		}
	}

	s.mu.Lock()
	s.entries, s.runs = staged, runs
	s.mu.Unlock()
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
