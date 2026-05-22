// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package mutation

import (
	"time"

	"pgregory.net/rapid"
)

// RandomDelay injects a uniform-random latency around any call.
// Surfaces timing-sensitive bugs (deadline races, timeout-handling
// gaps, ordering assumptions that rely on instantaneous execution).
// A law suite that doesn't catch RandomDelay-induced bugs is
// missing timeliness or ordering checks.
type RandomDelay struct {
	// Min and Max bound the injected latency in nanoseconds. The
	// operator draws a uniform sample in [Min, Max] and returns it.
	// Min must be ≥ 0; Max must be ≥ Min.
	Min time.Duration
	Max time.Duration
}

// Name returns the operator's stable identifier.
func (RandomDelay) Name() string { return "RandomDelay" }

// Delay returns the duration to sleep before (or after) the call.
// Returns zero when Min == Max == 0.
func (r RandomDelay) Delay(rt *rapid.T) time.Duration {
	if r.Max <= 0 || r.Max < r.Min {
		return 0
	}
	if r.Min == r.Max {
		return r.Min
	}
	span := int64(r.Max - r.Min)
	picked := rapid.Int64Range(0, span).Draw(rt, "RandomDelay_pick")
	return r.Min + time.Duration(picked)
}
