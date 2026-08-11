// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package sideeffecttest holds the generated harness and double for
// [go.thesmos.sh/testkit/conformance/corpus/iface/mixin/sideeffect], and the
// in-memory subject they are run against — scaffolding for the run, so it lives
// beside the harness rather than in the package stating the shape.
package sideeffecttest

import (
	"context"
	"errors"
	"sync"

	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/sideeffect"
)

// InMemory is the implementation the generated conformance harness is run
// against.
type InMemory struct {
	mu      sync.Mutex
	touches map[string]int
}

var _ sideeffect.Mixed = (*InMemory)(nil)

// NewInMemory returns an untouched subject.
func NewInMemory() *InMemory { return &InMemory{touches: map[string]int{}} }

// Touch reports only whether it failed, which is what makes the mixin
// necessary: the effect is out of band, so a subject that accepted the call and
// did nothing satisfies every check derived from this signature alone.
func (s *InMemory) Touch(ctx context.Context, key string) error {
	if err := contextErr(ctx); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.touches[key]++
	return nil
}

// Observed is what the mixin's observe parameter names, and the only reason the
// effect is checkable at all.
func (s *InMemory) Observed(ctx context.Context, key string) (int, error) {
	if err := contextErr(ctx); err != nil {
		return 0, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.touches[key], nil
}

// contextErr reports a cancelled or expired context, and tolerates a nil one.
//
// Nil is not a legal context and reaches production anyway, through a caller
// that forgot one. Returning an error is a failed request; dereferencing it is
// an outage, which is why the generated check asks only that this does not
// panic.
func contextErr(ctx context.Context) error {
	if ctx == nil {
		return errors.New("sideeffecttest: nil context")
	}
	return ctx.Err()
}
