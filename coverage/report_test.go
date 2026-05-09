// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package coverage_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/coverage"
)

func TestAggregatorReport(t *testing.T) {
	t.Parallel()

	t.Run("nil receiver returns zero-value report", func(t *testing.T) {
		t.Parallel()
		var a *coverage.Aggregator
		r := a.Report()
		testkit.Equal(t, len(r.Components), 0, "no components")
		testkit.True(t, r.Subsystem == nil, "no subsystem")
	})

	t.Run("renders components in alphabetical order with summaries", func(t *testing.T) {
		t.Parallel()
		a := coverage.NewAggregator()
		a.SetComponent("Z", &coverage.ComponentCoverage{
			StateSpace: coverage.StateSpaceMetrics{Explored: 10, Saturated: true},
			FireRate:   map[string]float64{"AUTO-A": 0.04, "AUTO-B": 0.5},
			REQToLaw:   map[string][]string{"REQ-1": {"AUTO-B"}},
			BranchHit:  coverage.BranchCoverageMetrics{Hit: 50, Total: 100},
		})
		a.SetComponent("A", &coverage.ComponentCoverage{
			FireRate: map[string]float64{"AUTO-X": 1.0},
		})

		r := a.Report()
		testkit.Equal(t, len(r.Components), 2, "two components")
		testkit.Equal(t, r.Components[0].Name, "A", "alphabetical: A first")
		testkit.Equal(t, r.Components[1].Name, "Z", "alphabetical: Z second")

		z := r.Components[1]
		testkit.Equal(t, z.ExploredStates, 10, "explored states")
		testkit.Equal(t, z.Saturated, true, "saturated flag")
		testkit.Equal(t, z.ActiveLaws, 2, "two laws active")
		testkit.Equal(t, z.TotalLaws, 2, "two laws registered")
		testkit.Equal(t, z.WeakLaws, []string{"AUTO-A"}, "AUTO-A is weak")
		testkit.Equal(t, z.REQsCovered, 1, "one REQ covered")
		testkit.Equal(t, z.BranchHitRatio, 0.5, "50%% branch coverage")
	})

	t.Run("nil ComponentCoverage entry produces empty summary", func(t *testing.T) {
		t.Parallel()
		a := coverage.NewAggregator()
		a.SetComponent("X", nil)
		r := a.Report()
		testkit.Equal(t, len(r.Components), 1, "one entry")
		testkit.Equal(t, r.Components[0].ExploredStates, 0, "zero values")
	})

	t.Run("populates Subsystem summary when present", func(t *testing.T) {
		t.Parallel()
		a := coverage.NewAggregator()
		a.Subsystem = &coverage.SubsystemCoverage{
			Components:        []string{"Ledger", "Scheduler"},
			SubsystemREQs:     map[string][]string{"REQ-1": {"INV-A"}, "REQ-2": {"INV-B"}},
			InvariantFireRate: map[string]float64{"INV-A": 0.5, "INV-B": 0},
			CrossComponentCausality: coverage.CausalityMetrics{
				EdgesObserved: 10, ComponentPairsLinked: 2,
			},
		}
		r := a.Report()
		testkit.True(t, r.Subsystem != nil, "subsystem summary populated")
		testkit.Equal(t, r.Subsystem.ComponentsTracked, 2, "2 components")
		testkit.Equal(t, r.Subsystem.ActiveInvariants, 1, "INV-A is active, INV-B is zero")
		testkit.Equal(t, r.Subsystem.TotalInvariants, 2, "2 invariants total")
		testkit.Equal(t, r.Subsystem.SubsystemREQsCovered, 2, "2 REQs")
		testkit.Equal(t, r.Subsystem.CausalityEdgesObserved, 10, "edges")
		testkit.Equal(t, r.Subsystem.ComponentPairsLinked, 2, "pairs")
	})
}

func TestAggregatorDiffSince(t *testing.T) {
	t.Parallel()

	t.Run("reports state-space additions and law fire-rate deltas", func(t *testing.T) {
		t.Parallel()
		prior := coverage.NewAggregator()
		prior.SetComponent("Ledger", &coverage.ComponentCoverage{
			StateSpace: coverage.StateSpaceMetrics{Explored: 100},
			FireRate:   map[string]float64{"AUTO-A": 0.5, "AUTO-B": 0.5},
			REQToLaw:   map[string][]string{"REQ-1": {"AUTO-A"}},
		})

		current := coverage.NewAggregator()
		current.SetComponent("Ledger", &coverage.ComponentCoverage{
			StateSpace: coverage.StateSpaceMetrics{Explored: 130},
			FireRate:   map[string]float64{"AUTO-A": 0.7, "AUTO-C": 0.3},
			REQToLaw:   map[string][]string{"REQ-1": {"AUTO-A"}, "REQ-2": {"AUTO-C"}},
		})

		d := current.DiffSince(prior)
		ledger := d.PerComponent["Ledger"]
		testkit.Equal(t, ledger.StatesAdded, 30, "30 new states")
		testkit.Equal(t, ledger.NewLaws, []string{"AUTO-C"}, "AUTO-C is new")
		testkit.Equal(t, ledger.LostLaws, []string{"AUTO-B"}, "AUTO-B lost")
		testkit.Equal(t, ledger.REQsAdded, []string{"REQ-2"}, "REQ-2 added")
		testkit.Equal(t, len(ledger.REQsRemoved), 0, "none removed")

		testkit.True(t, ledger.FireRateChanges["AUTO-A"] > 0.19 && ledger.FireRateChanges["AUTO-A"] < 0.21,
			"AUTO-A delta ~+0.2")
		testkit.True(t, ledger.FireRateChanges["AUTO-B"] < 0, "AUTO-B negative delta")
		testkit.Equal(t, ledger.FireRateChanges["AUTO-C"], 0.3, "AUTO-C delta = 0.3")
	})

	t.Run("nil prior treats as empty (every component is new)", func(t *testing.T) {
		t.Parallel()
		current := coverage.NewAggregator()
		current.SetComponent("Ledger", &coverage.ComponentCoverage{
			FireRate: map[string]float64{"AUTO-A": 1.0},
		})
		d := current.DiffSince(nil)
		ledger := d.PerComponent["Ledger"]
		testkit.Equal(t, ledger.NewLaws, []string{"AUTO-A"}, "law is new vs nil prior")
	})

	t.Run("component disappearing produces a diff entry with lost laws", func(t *testing.T) {
		t.Parallel()
		prior := coverage.NewAggregator()
		prior.SetComponent("Gone", &coverage.ComponentCoverage{
			FireRate: map[string]float64{"AUTO-X": 0.5},
			REQToLaw: map[string][]string{"REQ-1": {"AUTO-X"}},
		})
		current := coverage.NewAggregator()

		d := current.DiffSince(prior)
		gone := d.PerComponent["Gone"]
		testkit.Equal(t, gone.LostLaws, []string{"AUTO-X"}, "all laws lost")
		testkit.Equal(t, gone.REQsRemoved, []string{"REQ-1"}, "REQ removed")
		testkit.Equal(t, gone.FireRateChanges["AUTO-X"], -0.5, "fire rate dropped to zero")
	})

	t.Run("identical aggregators produce a no-op diff", func(t *testing.T) {
		t.Parallel()
		build := func() *coverage.Aggregator {
			a := coverage.NewAggregator()
			a.SetComponent("Ledger", &coverage.ComponentCoverage{
				StateSpace: coverage.StateSpaceMetrics{Explored: 10},
				FireRate:   map[string]float64{"AUTO-A": 0.5},
				REQToLaw:   map[string][]string{"REQ-1": {"AUTO-A"}},
			})
			return a
		}
		d := build().DiffSince(build())
		ledger := d.PerComponent["Ledger"]
		testkit.Equal(t, ledger.StatesAdded, 0, "no states added")
		testkit.Equal(t, len(ledger.NewLaws), 0, "no new laws")
		testkit.Equal(t, len(ledger.LostLaws), 0, "no lost laws")
		testkit.Equal(t, len(ledger.FireRateChanges), 0, "no rate deltas")
	})

	t.Run("subsystem diff fires when either side has a subsystem", func(t *testing.T) {
		t.Parallel()
		prior := coverage.NewAggregator()
		prior.Subsystem = &coverage.SubsystemCoverage{
			InvariantFireRate: map[string]float64{"INV-A": 0.5, "INV-B": 0.2},
			CrossComponentCausality: coverage.CausalityMetrics{
				EdgesObserved: 5, ComponentPairsLinked: 1,
			},
		}
		current := coverage.NewAggregator()
		current.Subsystem = &coverage.SubsystemCoverage{
			InvariantFireRate: map[string]float64{"INV-A": 0.7, "INV-C": 0.4},
			CrossComponentCausality: coverage.CausalityMetrics{
				EdgesObserved: 15, ComponentPairsLinked: 3,
			},
		}
		d := current.DiffSince(prior)
		testkit.True(t, d.Subsystem != nil, "subsystem diff present")
		testkit.Equal(t, d.Subsystem.NewInvariants, []string{"INV-C"}, "INV-C is new")
		testkit.Equal(t, d.Subsystem.LostInvariants, []string{"INV-B"}, "INV-B lost")
		testkit.Equal(t, d.Subsystem.CausalityEdgeDelta, 10, "edges +10")
		testkit.Equal(t, d.Subsystem.ComponentPairsDelta, 2, "pairs +2")
		testkit.True(t, d.Subsystem.InvariantFireRateChanges["INV-A"] > 0.19 &&
			d.Subsystem.InvariantFireRateChanges["INV-A"] < 0.21, "INV-A ~+0.2")
		testkit.Equal(t, d.Subsystem.InvariantFireRateChanges["INV-B"], -0.2, "INV-B -0.2")
		testkit.Equal(t, d.Subsystem.InvariantFireRateChanges["INV-C"], 0.4, "INV-C +0.4")
	})

	t.Run("absence of subsystem on both sides leaves Subsystem nil", func(t *testing.T) {
		t.Parallel()
		d := coverage.NewAggregator().DiffSince(coverage.NewAggregator())
		testkit.True(t, d.Subsystem == nil, "no subsystem entry")
	})

	t.Run("subsystem appearing on current side is treated as new", func(t *testing.T) {
		t.Parallel()
		prior := coverage.NewAggregator()
		current := coverage.NewAggregator()
		current.Subsystem = &coverage.SubsystemCoverage{
			InvariantFireRate: map[string]float64{"INV-A": 0.5},
			CrossComponentCausality: coverage.CausalityMetrics{
				EdgesObserved: 7, ComponentPairsLinked: 2,
			},
		}

		d := current.DiffSince(prior)
		testkit.True(t, d.Subsystem != nil, "subsystem diff present")
		testkit.Equal(t, d.Subsystem.NewInvariants, []string{"INV-A"}, "INV-A is new vs nil prior")
		testkit.Equal(t, len(d.Subsystem.LostInvariants), 0, "nothing lost")
		testkit.Equal(t, d.Subsystem.CausalityEdgeDelta, 7, "edges +7 vs zero")
		testkit.Equal(t, d.Subsystem.ComponentPairsDelta, 2, "pairs +2 vs zero")
		testkit.Equal(t, d.Subsystem.InvariantFireRateChanges["INV-A"], 0.5, "INV-A delta = 0.5")
	})

	t.Run("subsystem disappearing on current side is treated as lost", func(t *testing.T) {
		t.Parallel()
		prior := coverage.NewAggregator()
		prior.Subsystem = &coverage.SubsystemCoverage{
			InvariantFireRate: map[string]float64{"INV-A": 0.5},
			CrossComponentCausality: coverage.CausalityMetrics{
				EdgesObserved: 7, ComponentPairsLinked: 2,
			},
		}
		current := coverage.NewAggregator()

		d := current.DiffSince(prior)
		testkit.True(t, d.Subsystem != nil, "subsystem diff present")
		testkit.Equal(t, d.Subsystem.LostInvariants, []string{"INV-A"}, "INV-A lost")
		testkit.Equal(t, d.Subsystem.CausalityEdgeDelta, -7, "edges -7")
		testkit.Equal(t, d.Subsystem.ComponentPairsDelta, -2, "pairs -2")
		testkit.Equal(t, d.Subsystem.InvariantFireRateChanges["INV-A"], -0.5, "INV-A delta = -0.5")
	})

	t.Run("identical subsystems produce no fire-rate-change map", func(t *testing.T) {
		t.Parallel()
		build := func() *coverage.Aggregator {
			a := coverage.NewAggregator()
			a.Subsystem = &coverage.SubsystemCoverage{
				InvariantFireRate: map[string]float64{"INV-A": 0.5},
			}
			return a
		}
		d := build().DiffSince(build())
		testkit.True(t, d.Subsystem != nil, "subsystem entry present")
		testkit.Equal(t, len(d.Subsystem.InvariantFireRateChanges), 0, "no rate changes")
	})
}
