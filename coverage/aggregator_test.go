// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package coverage_test

import (
	"encoding/json"
	"strings"
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/coverage"
)

func TestNewAggregator(t *testing.T) {
	t.Parallel()

	t.Run("constructs with PerComponent ready to write", func(t *testing.T) {
		t.Parallel()
		a := coverage.NewAggregator()
		testkit.True(t, a.PerComponent != nil, "PerComponent map allocated")
		testkit.True(t, a.Subsystem == nil, "Subsystem starts nil")
	})
}

func TestAggregatorSetComponent(t *testing.T) {
	t.Parallel()

	t.Run("registers a component by name", func(t *testing.T) {
		t.Parallel()
		a := coverage.NewAggregator()
		c := &coverage.ComponentCoverage{StateSpace: coverage.StateSpaceMetrics{Explored: 7}}
		a.SetComponent("Ledger", c)
		testkit.Equal(t, a.PerComponent["Ledger"], c, "stored under the name")
	})

	t.Run("late registration after zero-value Aggregator allocates the map", func(t *testing.T) {
		t.Parallel()
		var a coverage.Aggregator
		a.SetComponent("Ledger", &coverage.ComponentCoverage{})
		testkit.True(t, a.PerComponent != nil, "lazy-allocates the map")
	})

	t.Run("panics on empty name", func(t *testing.T) {
		t.Parallel()
		a := coverage.NewAggregator()
		recovered := testkit.Panics(t, func() {
			a.SetComponent("", &coverage.ComponentCoverage{})
		}, "empty name must panic")
		testkit.Assert(t, asString(recovered)).Contains("name is empty", "diagnostic names the misuse")
	})
}

func TestAggregatorComponentNames(t *testing.T) {
	t.Parallel()

	t.Run("nil receiver returns nil", func(t *testing.T) {
		t.Parallel()
		var a *coverage.Aggregator
		testkit.Equal(t, len(a.ComponentNames()), 0, "nil receiver")
	})

	t.Run("returns sorted names", func(t *testing.T) {
		t.Parallel()
		a := coverage.NewAggregator()
		a.SetComponent("Z", &coverage.ComponentCoverage{})
		a.SetComponent("A", &coverage.ComponentCoverage{})
		a.SetComponent("M", &coverage.ComponentCoverage{})
		testkit.Equal(t, a.ComponentNames(), []string{"A", "M", "Z"}, "sorted")
	})
}

func TestAggregatorJSONRoundTrip(t *testing.T) {
	t.Parallel()

	t.Run("round-trips through encoding/json", func(t *testing.T) {
		t.Parallel()
		original := coverage.NewAggregator()
		original.SetComponent("Ledger", &coverage.ComponentCoverage{
			StateSpace: coverage.StateSpaceMetrics{Explored: 42, Saturated: true},
			FireRate:   map[string]float64{"AUTO-X": 0.8},
			REQToLaw:   map[string][]string{"REQ-1": {"AUTO-X"}},
			BranchHit:  coverage.BranchCoverageMetrics{Hit: 90, Total: 100},
		})
		original.Subsystem = &coverage.SubsystemCoverage{
			Components:        []string{"Ledger"},
			SubsystemREQs:     map[string][]string{"REQ-S": {"INV-Q"}},
			InvariantFireRate: map[string]float64{"INV-Q": 0.7},
			CrossComponentCausality: coverage.CausalityMetrics{
				EdgesObserved: 12, ComponentPairsLinked: 3,
			},
		}

		buf, err := json.Marshal(original)
		testkit.NoError(t, err, "marshal")

		var got coverage.Aggregator
		testkit.NoError(t, json.Unmarshal(buf, &got), "unmarshal")

		testkit.Equal(t, got.PerComponent["Ledger"].StateSpace.Explored, 42, "explored states preserved")
		testkit.Equal(t, got.PerComponent["Ledger"].FireRate["AUTO-X"], 0.8, "fire rate preserved")
		testkit.Equal(t, got.Subsystem.CrossComponentCausality.EdgesObserved, 12, "subsystem causality preserved")
	})
}

func asString(v any) string {
	switch x := v.(type) {
	case string:
		return x
	case error:
		return x.Error()
	default:
		return strings.TrimSpace(strings.ReplaceAll(strings.Trim(prettyAny(v), `"`), "\n", " "))
	}
}

func prettyAny(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		return ""
	}
	return string(b)
}
