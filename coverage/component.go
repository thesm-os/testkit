// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package coverage

import "sort"

// ComponentCoverage is the per-interface (or per-component-in-a-sim)
// signal bundle. The model runner emits one of these per interface
// it harnesses; sim builds one per registered component. The fields
// match INTEGRATION.md§Coverage aggregation and are populated by the
// runtime trackers that ship with their producing generator.
type ComponentCoverage struct {
	// StateSpace is the run's exploration footprint. Zero value
	// indicates state-space tracking was not enabled.
	StateSpace StateSpaceMetrics `json:"state_space"`

	// FireRate maps a law's stable ID (e.g., "AUTO-READ-AFTER-WRITE")
	// to its per-iteration fire rate in [0.0, 1.0]. A zero rate
	// means the law was registered but never fired; entries below
	// the runner's weak-law threshold surface in
	// [Report.WeakLaws]. Nil maps render as no laws tracked.
	FireRate map[string]float64 `json:"fire_rate,omitempty"`

	// REQToLaw maps a `//testkit:req REQ-...` ID to the laws that
	// cite it. Populated at codegen time and threaded through the
	// runner; the CI artifact at
	// `<artifactDir>/req-coverage-<run>.json` derives from this.
	REQToLaw map[string][]string `json:"req_to_law,omitempty"`

	// BranchHit is the Go-coverage delta produced during the run
	// when branch-coverage measurement was enabled (zero value
	// otherwise; see [BranchCoverageMetrics.Total]).
	BranchHit BranchCoverageMetrics `json:"branch_hit"`
}

// LawIDs returns every law ID present in [ComponentCoverage.FireRate]
// in sorted order. Used by aggregation and reporting code that
// must produce deterministic output.
func (c *ComponentCoverage) LawIDs() []string {
	if c == nil {
		return nil
	}
	out := make([]string, 0, len(c.FireRate))
	for id := range c.FireRate {
		out = append(out, id)
	}
	sort.Strings(out)
	return out
}

// REQIDs returns every REQ ID present in [ComponentCoverage.REQToLaw]
// in sorted order. Used by aggregation and reporting code that
// must produce deterministic output.
func (c *ComponentCoverage) REQIDs() []string {
	if c == nil {
		return nil
	}
	out := make([]string, 0, len(c.REQToLaw))
	for id := range c.REQToLaw {
		out = append(out, id)
	}
	sort.Strings(out)
	return out
}

// WeakLaws returns law IDs whose fire rate is strictly below
// threshold, in sorted order. Threshold defaults to 0.05 (5%) per
// model.md§Coverage signals; the consumer can override per call.
// Pass threshold <= 0 to use the default.
func (c *ComponentCoverage) WeakLaws(threshold float64) []string {
	if c == nil {
		return nil
	}
	if threshold <= 0 {
		threshold = 0.05
	}
	var out []string
	for _, id := range c.LawIDs() {
		if c.FireRate[id] < threshold {
			out = append(out, id)
		}
	}
	return out
}

// ActiveLawCount returns the count of laws whose fire rate is
// strictly positive. Used by [Report] to summarize "N active laws
// of M registered."
func (c *ComponentCoverage) ActiveLawCount() int {
	if c == nil {
		return 0
	}
	n := 0
	for _, rate := range c.FireRate {
		if rate > 0 {
			n++
		}
	}
	return n
}
