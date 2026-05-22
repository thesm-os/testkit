// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package mutation

import (
	"pgregory.net/rapid"
)

// LossyStream silently skips a fraction of StreamReader items.
// The wrapped iterator drops the item instead of yielding it.
// Surfaces missing stream-completeness contracts. A law suite that
// doesn't catch this is missing Stream-Reflects-Mutations,
// Stream-Permutation, or Stream-Over-Match (depending on shape).
type LossyStream struct {
	// Rate is the per-item drop probability in [0.0, 1.0].
	Rate float64
}

// Name returns the operator's stable identifier.
func (LossyStream) Name() string { return "LossyStream" }

// ShouldDrop reports whether the current stream item should be
// silently skipped (not yielded to the consumer).
func (l LossyStream) ShouldDrop(rt *rapid.T) bool {
	return fires(rt, "LossyStream_decision", l.Rate)
}
