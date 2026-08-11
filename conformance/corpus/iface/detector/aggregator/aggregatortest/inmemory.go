// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package aggregatortest holds the generated harness and double for
// [go.thesmos.sh/testkit/conformance/corpus/iface/detector/aggregator], and the
// in-memory subject they are run against.
package aggregatortest

import (
	"context"
	"errors"
	"sync"

	"go.thesmos.sh/testkit/conformance/corpus/iface/detector/aggregator"
)

// InMemory is the implementation the generated conformance harness is run
// against.
//
// It lives beside the harness rather than in the package declaring the
// interface, because that is what it is for: a fixture package states a shape,
// and the subject a conformance run holds to it is scaffolding for the run.
type InMemory struct {
	mu    sync.Mutex
	items []string
}

var _ aggregator.Aggregator = (*InMemory)(nil)

// NewInMemory returns an empty collection.
func NewInMemory() *InMemory { return &InMemory{} }

// Count reduces the whole collection to one number, which is the shape: there
// is no key to look up, only something to compute.
//
// It cannot miss. Nothing is passed in, so no caller — the harness included —
// can choose an input that makes it fail, which is why no "an error carries the
// zero value" check is generated for it.
func (s *InMemory) Count(ctx context.Context) (int, error) {
	if err := contextErr(ctx); err != nil {
		return 0, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.items), nil
}

// Add appends an item, so a test can observe a count other than zero.
func (s *InMemory) Add(item string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.items = append(s.items, item)
}

// contextErr reports a cancelled or expired context, and tolerates a nil one.
//
// Nil is not a legal context and reaches production anyway, through a caller
// that forgot one. Returning an error is a failed request; dereferencing it is
// an outage, which is why the generated check asks only that this does not
// panic.
func contextErr(ctx context.Context) error {
	if ctx == nil {
		return errors.New("aggregatortest: nil context")
	}
	return ctx.Err()
}
