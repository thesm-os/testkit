// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// provides the [AtomicCell] reference for the
// CompareAndSwap contract-tier shape. The cell stores a value
// keyed by a version token; writers race, the writer whose
// version matches the current value wins, and losers see the
// configured mismatch error.

package ref

import (
	"context"
	"sync"
)

// AtomicCell is a single-slot value with optimistic concurrency.
// Construct with [NewAtomicCell]. Thread-safe.
type AtomicCell[V any, Version comparable] struct {
	mu        sync.Mutex
	value     V
	version   Version
	present   bool
	mismatch  error
	versionOf func(V) Version
	nextVer   func(Version) Version
}

// NewAtomicCell constructs an [AtomicCell].
//
// versionOf extracts the version from a stored value (for read
// returns); nextVer produces the next version after a successful
// swap. mismatch is the error returned on a version conflict.
func NewAtomicCell[V any, Version comparable](
	versionOf func(V) Version,
	nextVer func(Version) Version,
	mismatch error,
) *AtomicCell[V, Version] {
	return &AtomicCell[V, Version]{
		mismatch:  mismatch,
		versionOf: versionOf,
		nextVer:   nextVer,
	}
}

// Get returns the current value and version, or zero values plus
// false when the cell is empty.
func (c *AtomicCell[V, Version]) Get(_ context.Context) (V, Version, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.present {
		var zV V
		var zVer Version
		return zV, zVer, false
	}
	return c.value, c.version, true
}

// CompareAndSwap accepts v iff the embedded version matches the
// stored version. On success the cell advances to nextVer(stored).
// On mismatch (including first-write race) the cell returns the
// mismatch error and the new value is not stored. The cell stores
// the initial value when empty regardless of version — empty is
// the start state, not a version-zero state.
func (c *AtomicCell[V, Version]) CompareAndSwap(_ context.Context, v V) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.present {
		c.value = v
		c.version = c.nextVer(c.versionOf(v))
		c.present = true
		return nil
	}
	if c.versionOf(v) != c.version {
		return c.mismatch
	}
	c.value = v
	c.version = c.nextVer(c.version)
	return nil
}
