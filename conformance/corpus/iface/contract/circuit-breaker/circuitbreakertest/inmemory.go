// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package circuitbreakertest holds the generated harness and double for
// [go.thesmos.sh/testkit/conformance/corpus/iface/contract/circuit-breaker],
// and the in-memory subject they are run against.
package circuitbreakertest

import (
	"context"
	"errors"
	"sync"

	circuitbreaker "go.thesmos.sh/testkit/conformance/corpus/iface/contract/circuit-breaker"
)

// ErrOpen reports a call refused because the breaker has tripped.
//
// Distinct from whatever the guarded call failed with, and that distinction is
// the contract: a caller seeing the downstream error retries the downstream,
// and a caller seeing this one backs off.
var ErrOpen = errors.New("circuitbreakertest: breaker is open")

// ErrDownstream is what the guarded call reports when it fails.
//
// Distinct from ErrOpen, and that distinction is the contract: a caller seeing
// this retries the downstream, and a caller seeing ErrOpen backs off.
var ErrDownstream = errors.New("circuitbreakertest: the downstream is unwell")

// UnwellKey names a downstream that always fails.
//
// The lever is the key rather than an exported switch, because a breaker guards
// something identified by the key it is called with — so "the downstream is
// unwell" is expressible in the interface's own terms, and a conformance run
// reaches it by asking for that downstream.
//
// It is not what the fixture derives, so the harness still seeds and smoke-runs
// against a healthy one. A subject that failed every call would fail its seed
// and never reach the check it exists for.
const UnwellKey = "unwell"

// InMemory is the implementation the generated conformance harness is run
// against.
//
// Failures are counted per key, because the threshold is about one downstream:
// a breaker counting across all of them takes a healthy dependency out of
// service because an unrelated one is struggling.
type InMemory struct {
	mu        sync.Mutex
	threshold int
	failures  map[string]int
}

var _ circuitbreaker.Contract = (*InMemory)(nil)

// NewInMemory returns a closed breaker that trips after threshold consecutive
// failures against one key.
func NewInMemory(threshold int) *InMemory {
	return &InMemory{threshold: threshold, failures: map[string]int{}}
}

// Run performs the guarded call unless the breaker has tripped for that key.
//
// A success resets the count rather than decrementing it. Consecutive is what
// the threshold means — a call that worked says the downstream is answering,
// and carrying older failures forward would trip on a pattern nobody saw.
func (s *InMemory) Run(ctx context.Context, key string) error {
	if err := contextErr(ctx); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.failures[key] >= s.threshold {
		return ErrOpen
	}
	if key == UnwellKey {
		s.failures[key]++
		return ErrDownstream
	}
	s.failures[key] = 0
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
		return errors.New("circuitbreakertest: nil context")
	}
	return ctx.Err()
}
