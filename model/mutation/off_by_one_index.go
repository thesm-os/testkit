// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package mutation

import (
	"pgregory.net/rapid"
)

// OffByOneIndex perturbs an Aggregator's count-style return by ±1
// with the configured rate. Models the classic boundary-condition
// bug. A law suite that doesn't catch this is missing a
// Count-Equals-Reference check.
type OffByOneIndex struct {
	// Rate is the per-call mutation probability in [0.0, 1.0].
	Rate float64
}

// Name returns the operator's stable identifier.
func (OffByOneIndex) Name() string { return "OffByOneIndex" }

// Mutate returns n perturbed by ±1 when the operator decides to
// mutate; otherwise returns n unchanged. The sign is drawn
// uniformly; +1 and -1 are equally likely.
func (o OffByOneIndex) Mutate(rt *rapid.T, n int) int {
	if !fires(rt, "OffByOneIndex_decision", o.Rate) {
		return n
	}
	if rapid.Bool().Draw(rt, "OffByOneIndex_sign") {
		return n + 1
	}
	return n - 1
}
