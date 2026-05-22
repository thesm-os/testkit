// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package shrinker

import (
	"slices"
	"sort"
)

// Step is one action in a recorded sequence with its data
// dependencies. The shrinker treats two steps as causally
// connected when one writes a name the other reads.
type Step struct {
	// Name is the human-readable identifier (e.g., "Put(k1)").
	Name string

	// Reads is the set of value names this step depends on. Empty
	// for steps that only write.
	Reads []string

	// Writes is the set of value names this step produces. Empty
	// for steps that only read.
	Writes []string
}

// CausalHullIndices returns the indices of [Step]s in the causal
// hull of failedAt: the failing step plus every step it
// transitively depends on through reads-of-writes.
//
// Indices are returned in original order. A failedAt outside
// [0, len(steps)) returns nil.
//
// The hull is computed by walking backward from failedAt: for each
// name the failing step reads, find the most-recent prior step
// that writes that name; mark it as in the hull; recurse on its
// reads. Steps with no causal connection to the failure are
// excluded.
func CausalHullIndices(steps []Step, failedAt int) []int {
	if failedAt < 0 || failedAt >= len(steps) {
		return nil
	}

	inHull := map[int]struct{}{failedAt: {}}
	// Worklist of indices whose Reads we still need to resolve.
	work := []int{failedAt}
	for len(work) > 0 {
		i := work[0]
		work = work[1:]
		for _, name := range steps[i].Reads {
			// Find the most-recent prior step that writes name.
			for j := i - 1; j >= 0; j-- {
				if slices.Contains(steps[j].Writes, name) {
					if _, already := inHull[j]; !already {
						inHull[j] = struct{}{}
						work = append(work, j)
					}
					break
				}
			}
		}
	}

	out := make([]int, 0, len(inHull))
	for i := range inHull {
		out = append(out, i)
	}
	sort.Ints(out)
	return out
}

// CausalHull returns the [Step] slice restricted to the failing
// step's causal hull, preserving original order. Wraps
// [CausalHullIndices] for the common shrink-and-return-steps case.
func CausalHull(steps []Step, failedAt int) []Step {
	idx := CausalHullIndices(steps, failedAt)
	if len(idx) == 0 {
		return nil
	}
	out := make([]Step, 0, len(idx))
	for _, i := range idx {
		out = append(out, steps[i])
	}
	return out
}

// ShrinkVerified returns the causal hull when the consumer-supplied
// verify callback confirms the shrunk sequence still reproduces the
// failure. When verify reports false, the original sequence is
// returned unchanged — the hull is incomplete, typically because a
// Step's Reads/Writes annotation missed a dependency.
//
// verify receives the candidate shrunk sequence and re-runs the
// SUT against it; returns true iff the failure persists. Callers
// who don't need re-run verification should use [CausalHull]
// directly.
func ShrinkVerified(steps []Step, failedAt int, verify func([]Step) bool) []Step {
	hull := CausalHull(steps, failedAt)
	if len(hull) == 0 {
		return steps
	}
	if !verify(hull) {
		// Hull was incomplete; preserve the original sequence.
		return steps
	}
	return hull
}
