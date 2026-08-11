// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package multiaggregatortest holds the generated harness and double for
// [go.thesmos.sh/testkit/conformance/corpus/iface/detector/multiaggregator], and
// the in-memory subject they are run against — scaffolding for the run, so it
// lives beside the harness rather than in the package stating the shape.
package multiaggregatortest

import (
	"context"
	"errors"
	"sync"

	"go.thesmos.sh/testkit/conformance/corpus/iface/detector/multiaggregator"
)

// InMemory is the implementation the generated conformance harness is run
// against.
type InMemory struct {
	mu    sync.Mutex
	items []int
}

var _ multiaggregator.MultiAggregator = (*InMemory)(nil)

// NewInMemory returns an empty collection.
func NewInMemory() *InMemory { return &InMemory{} }

// Stats reduces the collection to several numbers at once, which is the only
// thing separating this shape from the single aggregator.
//
// Like that one it takes nothing, so no caller can choose an input that makes it
// fail and no "an error carries the zero value" check is generated — even though
// there are two slots here that could disagree with the error. Both are zeroed
// anyway, and this package's own test is what says so.
func (s *InMemory) Stats(ctx context.Context) (int, int, error) {
	if err := contextErr(ctx); err != nil {
		return 0, 0, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	var sum int
	for _, n := range s.items {
		sum += n
	}
	return len(s.items), sum, nil
}

// Add appends an item, so a test can observe stats other than zero.
func (s *InMemory) Add(n int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.items = append(s.items, n)
}

// contextErr reports a cancelled or expired context, and tolerates a nil one.
//
// Nil is not a legal context and reaches production anyway, through a caller
// that forgot one. Returning an error is a failed request; dereferencing it is
// an outage, which is why the generated check asks only that this does not
// panic.
func contextErr(ctx context.Context) error {
	if ctx == nil {
		return errors.New("multiaggregatortest: nil context")
	}
	return ctx.Err()
}
