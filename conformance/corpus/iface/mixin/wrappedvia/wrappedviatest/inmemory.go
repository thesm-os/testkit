// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package wrappedviatest holds the generated harness and double for
// [go.thesmos.sh/testkit/conformance/corpus/iface/mixin/wrappedvia], and the
// in-memory subject they are run against.
package wrappedviatest

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/wrappedvia"
)

// ErrUnderlying is the cause Open wraps, and what Cause hands back.
//
// The mixin's claim is that the two answer to one another: a caller unwrapping
// what Open returned reaches what Cause reports, so an error can be handled by
// kind rather than by reading its text.
var ErrUnderlying = errors.New("wrappedviatest: underlying failure")

// failing is the name Open refuses, so the wrapping path is reachable.
const failing = "closed"

// InMemory is the implementation the generated conformance harness is run
// against.
type InMemory struct {
	mu   sync.Mutex
	last error
}

var _ wrappedvia.Mixed = (*InMemory)(nil)

// NewInMemory returns a subject that has failed at nothing.
func NewInMemory() *InMemory { return &InMemory{} }

// Open wraps rather than replacing, which is what makes errors.Is work through
// it: a caller two layers up asks about the cause and gets an answer.
func (s *InMemory) Open(ctx context.Context, name string) error {
	if err := contextErr(ctx); err != nil {
		return err
	}
	if name != failing {
		return nil
	}
	err := fmt.Errorf("opening %q: %w", name, ErrUnderlying)
	s.mu.Lock()
	defer s.mu.Unlock()
	s.last = ErrUnderlying
	return err
}

// Cause is the partner the mixin names through `fn=Cause`, and reports what the
// last failure wrapped.
func (s *InMemory) Cause(ctx context.Context) error {
	if err := contextErr(ctx); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.last
}

// FailingName is the name Open refuses.
func FailingName() string { return failing }

// contextErr reports a cancelled or expired context, and tolerates a nil one.
func contextErr(ctx context.Context) error {
	if ctx == nil {
		return errors.New("wrappedviatest: nil context")
	}
	return ctx.Err()
}
