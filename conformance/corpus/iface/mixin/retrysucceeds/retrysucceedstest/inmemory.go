// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package retrysucceedstest holds the generated harness and double for
// [go.thesmos.sh/testkit/conformance/corpus/iface/mixin/retrysucceeds], and the
// in-memory subject they are run against.
package retrysucceedstest

import (
	"context"
	"errors"
	"sync"

	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/retrysucceeds"
)

// ErrTransient is the failure a retry is expected to survive. Distinguishable
// from a permanent one on purpose: a caller that retried everything would loop
// on faults no repetition fixes.
var ErrTransient = errors.New("retrysucceedstest: transient failure")

// failuresBeforeSuccess is how many times Call fails before it succeeds, per
// key. Fixed rather than declared, because the mixin names no attempt count —
// which is why no check is generated for it.
const failuresBeforeSuccess = 2

// InMemory is the implementation the generated conformance harness is run
// against.
type InMemory struct {
	mu       sync.Mutex
	attempts map[string]int
}

var _ retrysucceeds.Mixed = (*InMemory)(nil)

// NewInMemory returns a subject that has been called for nothing.
func NewInMemory() *InMemory { return &InMemory{attempts: map[string]int{}} }

// Call fails transiently the first few times per key and then succeeds, which
// is what the mixin claims a caller can rely on.
func (s *InMemory) Call(ctx context.Context, key string) error {
	if err := contextErr(ctx); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.attempts[key]++
	if s.attempts[key] <= failuresBeforeSuccess {
		return ErrTransient
	}
	return nil
}

// Attempts reports how many calls were made, which is what makes "succeeds
// within N" observable at all — Call reports only this attempt's outcome.
func (s *InMemory) Attempts(ctx context.Context) (int, error) {
	if err := contextErr(ctx); err != nil {
		return 0, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	var total int
	for _, n := range s.attempts {
		total += n
	}
	return total, nil
}

// contextErr reports a cancelled or expired context, and tolerates a nil one.
func contextErr(ctx context.Context) error {
	if ctx == nil {
		return errors.New("retrysucceedstest: nil context")
	}
	return ctx.Err()
}
