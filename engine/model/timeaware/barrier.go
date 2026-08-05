// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package timeaware

import "sync"

// Barrier coordinates clock advancement with in-flight operations.
//
// Concurrent operations call [Barrier.Op] before starting and the
// returned release function when finished. Clock advancement calls
// [Barrier.Advance] with the advance closure; the closure runs
// under the barrier's write lock, so it blocks until every
// in-flight Op completes and prevents new Ops from starting until
// the advance returns.
//
// The barrier is safe for concurrent use. Op acquires a shared
// (read) lock; Advance acquires the exclusive (write) lock. This
// means multiple ops may proceed in parallel as long as no advance
// is pending; the moment an advance is requested, no new ops may
// start until the advance closes.
type Barrier struct {
	rw sync.RWMutex
}

// NewBarrier returns a fresh [Barrier].
func NewBarrier() *Barrier {
	return &Barrier{}
}

// Op marks the start of an in-flight operation. Returns the
// release closure the caller MUST invoke when the operation
// completes (typically via defer). Multiple Ops may be active
// concurrently.
func (b *Barrier) Op() func() {
	b.rw.RLock()
	released := false
	return func() {
		if released {
			return
		}
		released = true
		b.rw.RUnlock()
	}
}

// Advance runs the advance closure under the barrier's exclusive
// lock, blocking until every in-flight Op has released. After the
// closure returns, the barrier reopens for Ops.
//
// A nil advance is a no-op (the barrier still blocks-and-releases,
// useful as a synchronization fence).
func (b *Barrier) Advance(advance func()) {
	b.rw.Lock()
	defer b.rw.Unlock()
	if advance != nil {
		advance()
	}
}
