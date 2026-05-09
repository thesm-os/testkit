// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package coverage

import "sort"

// SubsystemCoverage is the sim-level aggregate, layered on top of
// the per-component coverage that sim's components contribute. Sim
// populates this directly: subsystem REQs come from
// `//testkit:sim` directives at the subsystem level, invariant fire
// rates from sim's invariant runner, and causality metrics from the
// cross-component DAG observation counted in
// [testkit/trace.Event.Causality].
type SubsystemCoverage struct {
	// Components lists the component names whose coverage is folded
	// into this subsystem report. Sorted in stable order.
	Components []string `json:"components,omitempty"`

	// SubsystemREQs maps a subsystem-level `//testkit:req REQ-...`
	// ID to the invariants that cite it. Distinct from per-component
	// REQ↔law mappings: subsystem REQs apply to invariants that span
	// components.
	SubsystemREQs map[string][]string `json:"subsystem_reqs,omitempty"`

	// InvariantFireRate maps a subsystem-invariant ID to its
	// per-iteration fire rate. Nil maps render as no invariants
	// tracked.
	InvariantFireRate map[string]float64 `json:"invariant_fire_rate,omitempty"`

	// CrossComponentCausality records DAG traversal at the
	// subsystem level. Zero value indicates causality tracking was
	// not enabled.
	CrossComponentCausality CausalityMetrics `json:"cross_component_causality"`
}

// InvariantIDs returns every invariant ID present in
// [SubsystemCoverage.InvariantFireRate] in sorted order. Used by
// aggregation and reporting code that must produce deterministic
// output.
func (s *SubsystemCoverage) InvariantIDs() []string {
	if s == nil {
		return nil
	}
	out := make([]string, 0, len(s.InvariantFireRate))
	for id := range s.InvariantFireRate {
		out = append(out, id)
	}
	sort.Strings(out)
	return out
}

// REQIDs returns every REQ ID present in
// [SubsystemCoverage.SubsystemREQs] in sorted order. Used by
// aggregation and reporting code that must produce deterministic
// output.
func (s *SubsystemCoverage) REQIDs() []string {
	if s == nil {
		return nil
	}
	out := make([]string, 0, len(s.SubsystemREQs))
	for id := range s.SubsystemREQs {
		out = append(out, id)
	}
	sort.Strings(out)
	return out
}

// ActiveInvariantCount returns the count of invariants whose fire
// rate is strictly positive. Used by [Report] to summarize "N active
// invariants of M registered."
func (s *SubsystemCoverage) ActiveInvariantCount() int {
	if s == nil {
		return 0
	}
	n := 0
	for _, rate := range s.InvariantFireRate {
		if rate > 0 {
			n++
		}
	}
	return n
}
