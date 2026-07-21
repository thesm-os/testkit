// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package model

import (
	"sort"
	"sync"

	"go.thesmos.sh/testkit/coverage"
)

// defaultSaturationThreshold is the number of consecutive iterations
// without a new state after which the state space is reported
// saturated (state space appears closed).
const defaultSaturationThreshold = 50

// stateSpaceTracker accumulates distinct state hashes across rapid
// iterations and reports the exploration footprint. It is the
// runtime producer for [coverage.StateSpaceMetrics]: the runner
// hashes the reference model's state after every iteration and feeds
// it here. Safe for the sequential rapid loop; the mutex guards
// against any incidental concurrent observation.
type stateSpaceTracker struct {
	mu        sync.Mutex
	seen      map[uint64]struct{}
	sinceNew  int
	threshold int
}

// newStateSpaceTracker constructs a tracker saturating after
// threshold repeat-only iterations. threshold <= 0 uses
// [defaultSaturationThreshold].
func newStateSpaceTracker(threshold int) *stateSpaceTracker {
	if threshold <= 0 {
		threshold = defaultSaturationThreshold
	}
	return &stateSpaceTracker{seen: make(map[uint64]struct{}), threshold: threshold}
}

// observe records one iteration's state hash and returns the metrics
// as of this observation. A previously-unseen hash grows the explored
// set and resets the saturation counter; a repeat advances it.
func (t *stateSpaceTracker) observe(h uint64) coverage.StateSpaceMetrics {
	t.mu.Lock()
	defer t.mu.Unlock()
	if _, ok := t.seen[h]; ok {
		t.sinceNew++
	} else {
		t.seen[h] = struct{}{}
		t.sinceNew = 0
	}
	return coverage.StateSpaceMetrics{
		Explored:               len(t.seen),
		IterationsSinceLastNew: t.sinceNew,
		Saturated:              t.sinceNew >= t.threshold,
	}
}

// buildREQToLaw builds the REQ-to-law matrix from a registry: each
// requirement ID mapped to the sorted IDs of the laws that cite it.
// Laws with no REQ tag are excluded. Returns nil for a nil registry.
// The runner fills [coverage.ComponentCoverage.REQToLaw] from this;
// the runtime fire-rate fill is a separate, not-yet-wired concern.
func buildREQToLaw[T any](r *Registry[T]) map[string][]string {
	if r == nil {
		return nil
	}
	out := make(map[string][]string)
	for _, l := range r.laws {
		req := l.REQID()
		if req == "" {
			continue
		}
		out[req] = append(out[req], l.ID())
	}
	for req := range out {
		sort.Strings(out[req])
	}
	return out
}
