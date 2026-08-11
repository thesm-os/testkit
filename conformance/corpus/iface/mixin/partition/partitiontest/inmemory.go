// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package partitiontest holds the generated harness and double for
// [go.thesmos.sh/testkit/conformance/corpus/iface/mixin/partition], and the
// in-memory subject they are run against — scaffolding for the run, so it lives
// beside the harness rather than in the package stating the shape.
package partitiontest

import (
	"context"
	"errors"
	"sync"

	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/partition"
)

// ErrNotFound is what Read reports for a key the partition does not hold.
var ErrNotFound = errors.New("partitiontest: not found")

// InMemory is the implementation the generated conformance harness is run
// against.
//
// Keyed by partition first, which is the shape: a store that hashed the two
// together would be indistinguishable here and wrong the moment a partition has
// to be dropped or listed.
type InMemory struct {
	mu    sync.Mutex
	parts map[string]map[string]string
}

var _ partition.Mixed = (*InMemory)(nil)

// NewInMemory returns an empty store.
func NewInMemory() *InMemory {
	return &InMemory{parts: map[string]map[string]string{}}
}

// Put writes into the named partition, creating it on first use.
func (s *InMemory) Put(ctx context.Context, part, key, value string) error {
	if err := contextErr(ctx); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.parts[part] == nil {
		s.parts[part] = map[string]string{}
	}
	s.parts[part][key] = value
	return nil
}

// Read observes isolation, which is why the mixin names it: Put reports only
// whether it failed, so nothing about where the value landed is visible without
// reading it back.
func (s *InMemory) Read(ctx context.Context, part, key string) (string, error) {
	if err := contextErr(ctx); err != nil {
		return "", err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	v, ok := s.parts[part][key]
	if !ok {
		return "", ErrNotFound
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
		return errors.New("partitiontest: nil context")
	}
	return ctx.Err()
}
