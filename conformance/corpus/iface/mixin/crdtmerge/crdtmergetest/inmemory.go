// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package crdtmergetest holds the generated harnesses and doubles for
// [go.thesmos.sh/testkit/conformance/corpus/iface/mixin/crdtmerge], and the
// in-memory subjects they are run against.
package crdtmergetest

import (
	"context"
	"errors"
	"slices"
	"sync"

	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/crdtmerge"
)

// InMemory is a grow-only set: merging is union, which is commutative,
// associative and idempotent — the three properties that make a merge
// order-independent and the mixin's whole content.
//
// Sorted on read so two replicas that converged report identically. Without it
// convergence would hold and be unobservable, since the comparison would fail
// on insertion order.
type InMemory struct {
	mu    sync.Mutex
	items map[string]struct{}
}

var (
	_ crdtmerge.Mixed   = (*InMemory)(nil)
	_ crdtmerge.Replica = (*InMemory)(nil)
)

// NewInMemory returns an empty replica.
func NewInMemory() *InMemory { return &InMemory{items: map[string]struct{}{}} }

// Add introduces the divergence a merge reconciles.
func (s *InMemory) Add(ctx context.Context, item string) error {
	if err := contextErr(ctx); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.items[item] = struct{}{}
	return nil
}

// Merge folds a peer in by union, so merging twice changes nothing and merging
// in either order arrives at the same set.
//
// The peer is read through the interface rather than reached into, which is
// what makes Replica a contract in its own right: a merge that type-asserted
// its peer would work only against replicas of its own kind.
func (s *InMemory) Merge(ctx context.Context, peer crdtmerge.Replica) error {
	if err := contextErr(ctx); err != nil {
		return err
	}
	if peer == nil {
		return nil
	}
	theirs, err := peer.Items(ctx)
	if err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, item := range theirs {
		s.items[item] = struct{}{}
	}
	return nil
}

// Items reports the set in a stable order.
func (s *InMemory) Items(ctx context.Context) ([]string, error) {
	if err := contextErr(ctx); err != nil {
		return nil, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]string, 0, len(s.items))
	for item := range s.items {
		out = append(out, item)
	}
	slices.Sort(out)
	return out, nil
}

// contextErr reports a cancelled or expired context, and tolerates a nil one.
func contextErr(ctx context.Context) error {
	if ctx == nil {
		return errors.New("crdtmergetest: nil context")
	}
	return ctx.Err()
}
