// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package mutation

import (
	"pgregory.net/rapid"
)

// MissDeletes makes a Deleter return nil without actually removing
// the entry — the SUT claims the delete succeeded while the value
// remains observable to subsequent Readers. A law suite that doesn't
// catch this is missing a Delete-Removes check.
type MissDeletes struct {
	// Rate is the per-call miss probability in [0.0, 1.0].
	Rate float64
}

// Name returns the operator's stable identifier.
func (MissDeletes) Name() string { return "MissDeletes" }

// ShouldMiss reports whether the current Delete call should
// silently no-op.
func (m MissDeletes) ShouldMiss(rt *rapid.T) bool {
	return fires(rt, "MissDeletes_decision", m.Rate)
}
