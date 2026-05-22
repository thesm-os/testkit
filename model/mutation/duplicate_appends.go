// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package mutation

import (
	"pgregory.net/rapid"
)

// DuplicateAppends makes an Appender write the same entry twice
// when the operator decides to dup. Surfaces missing dedup contracts
// and gap-free-offset checks. A law suite that doesn't catch this
// is missing Appender-Monotonic-Offsets or Append-Only-No-Drops.
type DuplicateAppends struct {
	// Rate is the per-call duplication probability in [0.0, 1.0].
	Rate float64
}

// Name returns the operator's stable identifier.
func (DuplicateAppends) Name() string { return "DuplicateAppends" }

// ShouldDuplicate reports whether the current Append call should
// be written twice (the second call uses the same entry as the
// first).
func (d DuplicateAppends) ShouldDuplicate(rt *rapid.T) bool {
	return fires(rt, "DuplicateAppends_decision", d.Rate)
}
