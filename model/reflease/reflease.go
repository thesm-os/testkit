// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package reflease provides the [Tracker] reference for the
// AcquireLease contract-tier shape. Tracker enforces the
// "double-acquire blocks, release returns to available, release
// of unheld lease errors" contract.
package reflease

import (
	"context"
	"sync"
)

// Tracker is a set of keys whose acquisition is tracked. Construct
// with [NewTracker]. Thread-safe.
type Tracker[K comparable] struct {
	mu      sync.Mutex
	held    map[K]struct{}
	heldErr error
	freeErr error
}

// NewTracker constructs a Tracker. heldErr is returned by Acquire
// when the key is already held; freeErr is returned by Release
// when the key is not currently held.
func NewTracker[K comparable](heldErr, freeErr error) *Tracker[K] {
	return &Tracker[K]{
		held:    make(map[K]struct{}),
		heldErr: heldErr,
		freeErr: freeErr,
	}
}

// Acquire takes the lease for k. Returns heldErr when k is
// already held.
func (t *Tracker[K]) Acquire(_ context.Context, k K) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	if _, held := t.held[k]; held {
		return t.heldErr
	}
	t.held[k] = struct{}{}
	return nil
}

// Release returns the lease for k. Returns freeErr when k is
// not currently held (release-of-unheld is a usage error).
func (t *Tracker[K]) Release(_ context.Context, k K) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	if _, held := t.held[k]; !held {
		return t.freeErr
	}
	delete(t.held, k)
	return nil
}

// IsHeld reports whether k is currently leased.
func (t *Tracker[K]) IsHeld(k K) bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	_, held := t.held[k]
	return held
}
