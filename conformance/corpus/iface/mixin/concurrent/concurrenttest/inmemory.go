// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package concurrenttest holds the generated harness and double for
// [go.thesmos.sh/testkit/conformance/corpus/iface/mixin/concurrent], and the
// in-memory subject they are run against.
package concurrenttest

import (
	"context"
	"errors"
	"sync"

	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/concurrent"
)

// InMemory is the implementation the generated conformance harness is run
// against.
type InMemory struct {
	mu     sync.Mutex
	counts map[string]int
}

var _ concurrent.Mixed = (*InMemory)(nil)

// NewInMemory returns an empty counter set.
func NewInMemory() *InMemory { return &InMemory{counts: map[string]int{}} }

// Bump increments under the lock, so concurrent callers cannot lose an update
// between a read and a write. Losing one is invisible to every check derived
// from the signature: the call still succeeds and Count still answers.
func (s *InMemory) Bump(ctx context.Context, key string) error {
	if err := contextErr(ctx); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.counts[key]++
	return nil
}

// Count totals every key.
func (s *InMemory) Count(ctx context.Context) (int, error) {
	if err := contextErr(ctx); err != nil {
		return 0, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	var total int
	for _, n := range s.counts {
		total += n
	}
	return total, nil
}

// contextErr reports a cancelled or expired context, and tolerates a nil one.
func contextErr(ctx context.Context) error {
	if ctx == nil {
		return errors.New("concurrenttest: nil context")
	}
	return ctx.Err()
}
