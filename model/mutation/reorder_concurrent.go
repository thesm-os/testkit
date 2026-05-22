// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package mutation

import (
	"runtime"

	"pgregory.net/rapid"
)

// ReorderConcurrent yields the goroutine at random points before a
// call returns, perturbing the scheduler's natural ordering. Used
// to surface ordering-sensitive bugs in concurrent histories — a
// law suite that doesn't catch reorder-induced linearizability
// failures is missing the linearizability or causal-ordering check.
type ReorderConcurrent struct {
	// Rate is the per-call yield probability in [0.0, 1.0].
	Rate float64
}

// Name returns the operator's stable identifier.
func (ReorderConcurrent) Name() string { return "ReorderConcurrent" }

// MaybeYield yields the current goroutine (via runtime.Gosched)
// with [ReorderConcurrent.Rate] probability. Returns true when a
// yield was issued.
func (r ReorderConcurrent) MaybeYield(rt *rapid.T) bool {
	if !fires(rt, "ReorderConcurrent_decision", r.Rate) {
		return false
	}
	runtime.Gosched()
	return true
}
