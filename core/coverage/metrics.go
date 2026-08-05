// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package coverage

// StateSpaceMetrics records what was reached during a run. The model
// runner populates Explored from a Hash(state)-keyed visited set;
// IterationsSinceLastNew tracks how long the explored set has been
// stable, and Saturated is the consumer-decided threshold flag (the
// runner sets it once `IterationsSinceLastNew >= saturationThreshold`).
type StateSpaceMetrics struct {
	// Explored is the count of distinct states observed.
	Explored int `json:"explored"`

	// IterationsSinceLastNew is the gap, in iterations, since a
	// previously-unseen state was last reached. Zero when the most
	// recent iteration produced a new state. Used by the runner to
	// compute Saturated.
	IterationsSinceLastNew int `json:"iterations_since_last_new"`

	// Saturated reports that the state space appears closed: the
	// explored set has not grown for IterationsSinceLastNew
	// iterations, exceeding the runner's configured threshold. Used
	// by the coverage-summary header to surface "Saturation reached
	// at N states (state space appears closed)".
	Saturated bool `json:"saturated"`
}

// BranchCoverageMetrics carries the branch-coverage delta from Go's
// testing.Coverage hook. Total is zero when the runner did not enable
// branch-coverage measurement (Hit's ratio is then undefined; consumer
// reports "—").
type BranchCoverageMetrics struct {
	// Hit is the count of branches the run executed.
	Hit int `json:"hit"`

	// Total is the count of branches in the instrumented binary.
	// Zero when branch-coverage measurement was not enabled for this
	// run.
	Total int `json:"total"`
}

// Ratio reports Hit/Total, or zero when Total is zero. Callers
// distinguish "0% coverage" from "branch coverage disabled" via
// [BranchCoverageMetrics.Total].
func (b BranchCoverageMetrics) Ratio() float64 {
	if b.Total == 0 {
		return 0
	}
	return float64(b.Hit) / float64(b.Total)
}

// CausalityMetrics records traversal of the cross-component causality
// DAG at sim runtime. Sim populates EdgesObserved from
// [trace.Event.Causality] entries; ComponentPairsLinked counts the
// distinct (producer-component, consumer-component) pairs the run
// touched. Both contribute to the subsystem-level coverage report.
type CausalityMetrics struct {
	// EdgesObserved is the count of happens-before edges the run
	// traversed in the cross-component DAG.
	EdgesObserved int `json:"edges_observed"`

	// ComponentPairsLinked is the count of distinct
	// (producer, consumer) component pairs an edge connected. A
	// rising count widens the cross-component coverage; a falling
	// count surfaces in [DiffReport] as a regression signal.
	ComponentPairsLinked int `json:"component_pairs_linked"`
}
