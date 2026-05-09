// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package coverage

import "sort"

// Report is the snapshot view produced by [Aggregator.Report]. It is
// the shape the coverage-summary header lines, the PR-bot comment,
// and the `<artifactDir>/coverage-<run>.json` artifact serialize
// from. Every list is sorted in stable order so two equivalent
// aggregators emit byte-identical reports.
type Report struct {
	// Components is one summary per registered component, sorted by
	// component name.
	Components []ComponentSummary `json:"components,omitempty"`

	// Subsystem is the subsystem-level summary. Nil when the
	// underlying aggregator has no [SubsystemCoverage] (per-interface
	// model harnesses).
	Subsystem *SubsystemSummary `json:"subsystem,omitempty"`
}

// ComponentSummary is the per-component view inside a [Report].
type ComponentSummary struct {
	Name           string   `json:"name"`
	ExploredStates int      `json:"explored_states"`
	Saturated      bool     `json:"saturated"`
	ActiveLaws     int      `json:"active_laws"`
	TotalLaws      int      `json:"total_laws"`
	WeakLaws       []string `json:"weak_laws,omitempty"`
	REQsCovered    int      `json:"reqs_covered"`
	BranchHitRatio float64  `json:"branch_hit_ratio"`
}

// SubsystemSummary is the subsystem-level view inside a [Report].
type SubsystemSummary struct {
	ComponentsTracked      int `json:"components_tracked"`
	ActiveInvariants       int `json:"active_invariants"`
	TotalInvariants        int `json:"total_invariants"`
	SubsystemREQsCovered   int `json:"subsystem_reqs_covered"`
	CausalityEdgesObserved int `json:"causality_edges_observed"`
	ComponentPairsLinked   int `json:"component_pairs_linked"`
}

func summarizeComponent(name string, c *ComponentCoverage) ComponentSummary {
	cs := ComponentSummary{Name: name}
	if c == nil {
		return cs
	}
	cs.ExploredStates = c.StateSpace.Explored
	cs.Saturated = c.StateSpace.Saturated
	cs.ActiveLaws = c.ActiveLawCount()
	cs.TotalLaws = len(c.FireRate)
	cs.WeakLaws = c.WeakLaws(0)
	cs.REQsCovered = len(c.REQToLaw)
	cs.BranchHitRatio = c.BranchHit.Ratio()
	return cs
}

// summarizeSubsystem produces a per-subsystem summary. The caller
// (Aggregator.Report) guards against a nil Subsystem before calling,
// so this helper assumes a non-nil receiver.
func summarizeSubsystem(s *SubsystemCoverage) SubsystemSummary {
	return SubsystemSummary{
		ComponentsTracked:      len(s.Components),
		ActiveInvariants:       s.ActiveInvariantCount(),
		TotalInvariants:        len(s.InvariantFireRate),
		SubsystemREQsCovered:   len(s.SubsystemREQs),
		CausalityEdgesObserved: s.CrossComponentCausality.EdgesObserved,
		ComponentPairsLinked:   s.CrossComponentCausality.ComponentPairsLinked,
	}
}

// DiffReport is the delta produced by [Aggregator.DiffSince]. Each
// entry surfaces what changed between two aggregators; the absence
// of a component from prior or current produces a "joined" or "left"
// diff with the missing side rendered as zero-valued coverage.
type DiffReport struct {
	// PerComponent keys every component that appeared in prior or
	// current; entries describe what changed for that component.
	PerComponent map[string]ComponentDiff `json:"per_component,omitempty"`

	// Subsystem describes subsystem-level changes; nil when neither
	// aggregator has subsystem data.
	Subsystem *SubsystemDiff `json:"subsystem,omitempty"`
}

// ComponentDiff is the per-component delta inside a [DiffReport].
type ComponentDiff struct {
	StatesAdded     int                `json:"states_added"`
	NewLaws         []string           `json:"new_laws,omitempty"`
	LostLaws        []string           `json:"lost_laws,omitempty"`
	FireRateChanges map[string]float64 `json:"fire_rate_changes,omitempty"`
	REQsAdded       []string           `json:"reqs_added,omitempty"`
	REQsRemoved     []string           `json:"reqs_removed,omitempty"`
}

// SubsystemDiff is the subsystem-level delta inside a [DiffReport].
type SubsystemDiff struct {
	NewInvariants            []string           `json:"new_invariants,omitempty"`
	LostInvariants           []string           `json:"lost_invariants,omitempty"`
	InvariantFireRateChanges map[string]float64 `json:"invariant_fire_rate_changes,omitempty"`
	CausalityEdgeDelta       int                `json:"causality_edge_delta"`
	ComponentPairsDelta      int                `json:"component_pairs_delta"`
}

func diffComponent(prior, current *ComponentCoverage) ComponentDiff {
	d := ComponentDiff{}
	d.StatesAdded = currentStates(current) - currentStates(prior)

	priorLaws := lawSet(prior)
	currentLaws := lawSet(current)
	d.NewLaws = sortedDiff(currentLaws, priorLaws)
	d.LostLaws = sortedDiff(priorLaws, currentLaws)

	d.FireRateChanges = fireRateDeltas(prior, current)

	priorREQs := reqSet(prior)
	currentREQs := reqSet(current)
	d.REQsAdded = sortedDiff(currentREQs, priorREQs)
	d.REQsRemoved = sortedDiff(priorREQs, currentREQs)
	return d
}

func diffSubsystem(prior, current *SubsystemCoverage) SubsystemDiff {
	d := SubsystemDiff{}
	priorInv := invariantSet(prior)
	currentInv := invariantSet(current)
	d.NewInvariants = sortedDiff(currentInv, priorInv)
	d.LostInvariants = sortedDiff(priorInv, currentInv)
	d.InvariantFireRateChanges = invariantFireRateDeltas(prior, current)
	d.CausalityEdgeDelta = causalityEdges(current) - causalityEdges(prior)
	d.ComponentPairsDelta = componentPairs(current) - componentPairs(prior)
	return d
}

func currentStates(c *ComponentCoverage) int {
	if c == nil {
		return 0
	}
	return c.StateSpace.Explored
}

func lawSet(c *ComponentCoverage) map[string]struct{} {
	if c == nil {
		return nil
	}
	out := make(map[string]struct{}, len(c.FireRate))
	for id := range c.FireRate {
		out[id] = struct{}{}
	}
	return out
}

func reqSet(c *ComponentCoverage) map[string]struct{} {
	if c == nil {
		return nil
	}
	out := make(map[string]struct{}, len(c.REQToLaw))
	for id := range c.REQToLaw {
		out[id] = struct{}{}
	}
	return out
}

func invariantSet(s *SubsystemCoverage) map[string]struct{} {
	if s == nil {
		return nil
	}
	out := make(map[string]struct{}, len(s.InvariantFireRate))
	for id := range s.InvariantFireRate {
		out[id] = struct{}{}
	}
	return out
}

func causalityEdges(s *SubsystemCoverage) int {
	if s == nil {
		return 0
	}
	return s.CrossComponentCausality.EdgesObserved
}

func componentPairs(s *SubsystemCoverage) int {
	if s == nil {
		return 0
	}
	return s.CrossComponentCausality.ComponentPairsLinked
}

func fireRateDeltas(prior, current *ComponentCoverage) map[string]float64 {
	deltas := map[string]float64{}
	priorRates := rateMap(prior)
	currentRates := rateMap(current)
	for id, cur := range currentRates {
		p := priorRates[id]
		if cur != p {
			deltas[id] = cur - p
		}
	}
	for id, p := range priorRates {
		if _, ok := currentRates[id]; ok {
			continue
		}
		deltas[id] = -p
	}
	if len(deltas) == 0 {
		return nil
	}
	return deltas
}

func invariantFireRateDeltas(prior, current *SubsystemCoverage) map[string]float64 {
	deltas := map[string]float64{}
	priorRates := invariantRateMap(prior)
	currentRates := invariantRateMap(current)
	for id, cur := range currentRates {
		p := priorRates[id]
		if cur != p {
			deltas[id] = cur - p
		}
	}
	for id, p := range priorRates {
		if _, ok := currentRates[id]; ok {
			continue
		}
		deltas[id] = -p
	}
	if len(deltas) == 0 {
		return nil
	}
	return deltas
}

func rateMap(c *ComponentCoverage) map[string]float64 {
	if c == nil {
		return nil
	}
	return c.FireRate
}

func invariantRateMap(s *SubsystemCoverage) map[string]float64 {
	if s == nil {
		return nil
	}
	return s.InvariantFireRate
}

// sortedDiff returns the sorted keys present in a but absent from b.
func sortedDiff(a, b map[string]struct{}) []string {
	var out []string
	for id := range a {
		if _, ok := b[id]; !ok {
			out = append(out, id)
		}
	}
	sort.Strings(out)
	return out
}
