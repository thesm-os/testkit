// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// provides the [LeaseTracker] reference for the
// AcquireLease contract-tier shape. LeaseTracker enforces the
// "double-acquire blocks, release returns to available, release
// of unheld lease errors" contract.

package ref

import (
	"context"
	"sync"
)

// LeaseTracker is a set of keys whose acquisition is tracked. Construct
// with [NewLeaseTracker]. Thread-safe.
type LeaseTracker[K comparable] struct {
	mu      sync.Mutex
	held    map[K]struct{}
	heldErr error
	freeErr error
}

// NewLeaseTracker constructs a LeaseTracker. heldErr is returned by Acquire
// when the key is already held; freeErr is returned by Release
// when the key is not currently held.
func NewLeaseTracker[K comparable](heldErr, freeErr error) *LeaseTracker[K] {
	return &LeaseTracker[K]{
		held:    make(map[K]struct{}),
		heldErr: heldErr,
		freeErr: freeErr,
	}
}

// Acquire takes the lease for k. Returns heldErr when k is
// already held.
func (t *LeaseTracker[K]) Acquire(_ context.Context, k K) error {
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
func (t *LeaseTracker[K]) Release(_ context.Context, k K) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	if _, held := t.held[k]; !held {
		return t.freeErr
	}
	delete(t.held, k)
	return nil
}

// IsHeld reports whether k is currently leased.
func (t *LeaseTracker[K]) IsHeld(k K) bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	_, held := t.held[k]
	return held
}
