// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package causalchaintest holds the generated harness and double for
// [go.thesmos.sh/testkit/conformance/corpus/iface/composite/causal-chain],
// and the in-memory subject they are run against.
package causalchaintest

import (
	"context"
	"errors"
	"slices"
	"sync"

	causalchain "go.thesmos.sh/testkit/conformance/corpus/iface/composite/causal-chain"
)

// ErrUnmetDependency is what Append reports for an entry whose dependencies
// have not landed — the refusal that is the admission policy.
var ErrUnmetDependency = errors.New("causalchaintest: a dependency has not landed")

// InMemory is the implementation the generated conformance harness is run
// against: appends land in order, and only after their dependencies, so the
// verbatim replay respects causality by construction.
type InMemory struct {
	mu      sync.Mutex
	entries []causalchain.Entry
	landed  map[string]bool
}

var _ causalchain.Log = (*InMemory)(nil)

// NewInMemory returns an empty log.
func NewInMemory() *InMemory { return &InMemory{landed: map[string]bool{}} }

// Append admits the entry when every dependency has landed, and refuses it
// whole otherwise.
func (s *InMemory) Append(ctx context.Context, e causalchain.Entry) error {
	if err := contextErr(ctx); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if slices.ContainsFunc(e.DependsOn, func(dep string) bool { return !s.landed[dep] }) {
		return ErrUnmetDependency
	}
	s.entries = append(s.entries, e)
	s.landed[e.ID] = true
	return nil
}

// Replay answers the log verbatim — admission order is causal order here,
// which is exactly the claim the replay law walks.
func (s *InMemory) Replay(ctx context.Context) ([]causalchain.Entry, error) {
	if err := contextErr(ctx); err != nil {
		return nil, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return slices.Clone(s.entries), nil
}

// contextErr reports a cancelled or expired context, and tolerates a nil one.
func contextErr(ctx context.Context) error {
	if ctx == nil {
		return errors.New("causalchaintest: nil context")
	}
	return ctx.Err()
}
