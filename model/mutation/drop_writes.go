// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package mutation

import (
	"pgregory.net/rapid"
)

// DropWrites silently discards a configurable fraction of Writer
// or Mutator calls. The wrapped call returns nil (no error) when
// the operator decides to drop, leaving the SUT's state unchanged.
// A law suite that doesn't catch this is missing post-write
// observability — typically a missing Read-after-Write or
// Write-Observable check.
type DropWrites struct {
	// Rate is the per-call drop probability in [0.0, 1.0]. A rate
	// of 1.0 drops every call; 0.0 drops none.
	Rate float64
}

// Name returns the operator's stable identifier.
func (DropWrites) Name() string { return "DropWrites" }

// ShouldDrop reports whether the current Writer/Mutator call
// should be silently discarded.
func (d DropWrites) ShouldDrop(rt *rapid.T) bool {
	return fires(rt, "DropWrites_decision", d.Rate)
}
