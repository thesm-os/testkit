// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package serializabletest holds the generated harness and double for
// [go.thesmos.sh/testkit/conformance/corpus/iface/mixin/serializable], and the
// in-memory subject they are run against — scaffolding for the run, so it
// lives beside the harness rather than in the package stating the shape.
package serializabletest

import (
	"context"
	"errors"
	"slices"
	"sync"

	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/serializable"
)

// ErrNotFound is what Get reports for a key nothing recorded.
var ErrNotFound = errors.New("serializabletest: not found")

// InMemory is the implementation the generated conformance harness is run
// against.
type InMemory struct {
	mu      sync.Mutex
	entries []serializable.Entry
}

var _ serializable.Mixed = (*InMemory)(nil)

// NewInMemory returns an empty history.
func NewInMemory() *InMemory { return &InMemory{} }

// Record appends one operation.
func (s *InMemory) Record(ctx context.Context, e serializable.Entry) error {
	if err := contextErr(ctx); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.entries = append(s.entries, e)
	return nil
}

// History reports the recorded operations.
//
// A copy rather than the slice itself: an anomaly check walks the history
// while the subject may still be recording, and handing out the backing array
// would let a later append be observed mid-walk.
func (s *InMemory) History(ctx context.Context) ([]serializable.Entry, error) {
	if err := contextErr(ctx); err != nil {
		return nil, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return slices.Clone(s.entries), nil
}

// Get answers the latest recorded entry for a key, and reports a miss with
// the zero value — the read half the anomaly law instantiates its key at.
func (s *InMemory) Get(ctx context.Context, key string) (serializable.Entry, error) {
	if err := contextErr(ctx); err != nil {
		return serializable.Entry{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, e := range slices.Backward(s.entries) {
		if e.Key == key {
			return e, nil
		}
	}
	return serializable.Entry{}, ErrNotFound
}

// contextErr reports a cancelled or expired context, and tolerates a nil one.
func contextErr(ctx context.Context) error {
	if ctx == nil {
		return errors.New("serializabletest: nil context")
	}
	return ctx.Err()
}
