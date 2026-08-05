// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package coverage

import "sort"

// Aggregator composes per-component coverage with subsystem-level
// coverage into one report-and-diff surface. The model runner
// populates one [ComponentCoverage] per interface; sim's runner adds
// the [SubsystemCoverage]. CI tooling, the PR bot, and the
// `testkit <generator> coverage` subcommand all read through this
// type.
//
// The Aggregator's fields are exported so producers can populate
// directly; callers are expected to construct via [NewAggregator]
// (which initializes the maps) and then assign components via
// [Aggregator.SetComponent] or by writing into [Aggregator.PerComponent].
type Aggregator struct {
	// PerComponent maps a component or interface name to its
	// coverage bundle. Stable across the aggregator's lifetime;
	// later writes overwrite earlier entries.
	PerComponent map[string]*ComponentCoverage `json:"per_component,omitempty"`

	// Subsystem is the subsystem-level overlay. Nil when the
	// generator producing the aggregator is per-interface (model);
	// populated for sim and downstream sim-composed generators
	// (chaos, diff-rollout, replay).
	Subsystem *SubsystemCoverage `json:"subsystem,omitempty"`
}

// NewAggregator returns an [Aggregator] with PerComponent
// pre-allocated. Subsystem is left nil for the per-interface case;
// sim and downstream generators set it via direct assignment.
func NewAggregator() *Aggregator {
	return &Aggregator{
		PerComponent: make(map[string]*ComponentCoverage),
	}
}

// SetComponent registers (or replaces) the coverage for the named
// component. Panics on an empty name — coverage entries always have
// a stable key, and a silent fallback would corrupt the aggregation.
func (a *Aggregator) SetComponent(name string, c *ComponentCoverage) {
	if name == "" {
		//nolint:forbidigo // unrecoverable misuse: a coverage entry
		// without a stable key cannot be reported or diffed against.
		panic("coverage.Aggregator.SetComponent: name is empty")
	}
	if a.PerComponent == nil {
		a.PerComponent = make(map[string]*ComponentCoverage)
	}
	a.PerComponent[name] = c
}

// ComponentNames returns the names registered in
// [Aggregator.PerComponent] in sorted order. Used by [Aggregator.Report]
// and [Aggregator.DiffSince] to ensure deterministic output.
func (a *Aggregator) ComponentNames() []string {
	if a == nil {
		return nil
	}
	out := make([]string, 0, len(a.PerComponent))
	for name := range a.PerComponent {
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}

// Report produces a summary of every component plus the subsystem.
// The result is deterministic for a given Aggregator state — map
// iteration is normalized by sorting keys.
func (a *Aggregator) Report() Report {
	if a == nil {
		return Report{}
	}
	r := Report{
		Components: make([]ComponentSummary, 0, len(a.PerComponent)),
	}
	for _, name := range a.ComponentNames() {
		c := a.PerComponent[name]
		r.Components = append(r.Components, summarizeComponent(name, c))
	}
	if a.Subsystem != nil {
		s := summarizeSubsystem(a.Subsystem)
		r.Subsystem = &s
	}
	return r
}

// DiffSince compares this aggregator against a prior one and returns
// the delta. Useful for CI gates that fail on coverage regression
// and for the `testkit <generator> coverage --since <commit>` CLI
// flag.
func (a *Aggregator) DiffSince(prior *Aggregator) DiffReport {
	out := DiffReport{
		PerComponent: map[string]ComponentDiff{},
	}
	var priorMap map[string]*ComponentCoverage
	var priorSub *SubsystemCoverage
	if prior != nil {
		priorMap = prior.PerComponent
		priorSub = prior.Subsystem
	}

	for _, name := range mergedNames(a.PerComponent, priorMap) {
		out.PerComponent[name] = diffComponent(priorMap[name], a.PerComponent[name])
	}

	if a.Subsystem != nil || priorSub != nil {
		d := diffSubsystem(priorSub, a.Subsystem)
		out.Subsystem = &d
	}
	return out
}

// mergedNames returns the sorted union of keys across two maps. Both
// inputs are deduplicated by their map nature; the merge produces one
// entry per distinct key.
func mergedNames(a, b map[string]*ComponentCoverage) []string {
	seen := make(map[string]struct{}, len(a)+len(b))
	for name := range a {
		seen[name] = struct{}{}
	}
	for name := range b {
		seen[name] = struct{}{}
	}
	out := make([]string, 0, len(seen))
	for name := range seen {
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}
