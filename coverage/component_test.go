// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package coverage_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/coverage"
)

func TestComponentCoverageAccessors(t *testing.T) {
	t.Parallel()

	t.Run("nil receiver returns nil/zero across the surface", func(t *testing.T) {
		t.Parallel()
		var c *coverage.ComponentCoverage
		testkit.Equal(t, len(c.LawIDs()), 0, "nil LawIDs")
		testkit.Equal(t, len(c.REQIDs()), 0, "nil REQIDs")
		testkit.Equal(t, len(c.WeakLaws(0.05)), 0, "nil WeakLaws")
		testkit.Equal(t, c.ActiveLawCount(), 0, "nil ActiveLawCount")
	})

	t.Run("LawIDs returns sorted law IDs", func(t *testing.T) {
		t.Parallel()
		c := &coverage.ComponentCoverage{
			FireRate: map[string]float64{
				"AUTO-Z": 0.5,
				"AUTO-A": 1.0,
				"AUTO-M": 0.0,
			},
		}
		testkit.Equal(t, c.LawIDs(), []string{"AUTO-A", "AUTO-M", "AUTO-Z"}, "sorted")
	})

	t.Run("REQIDs returns sorted REQ IDs", func(t *testing.T) {
		t.Parallel()
		c := &coverage.ComponentCoverage{
			REQToLaw: map[string][]string{
				"REQ-Z": {"AUTO-X"},
				"REQ-A": {"AUTO-Y"},
			},
		}
		testkit.Equal(t, c.REQIDs(), []string{"REQ-A", "REQ-Z"}, "sorted")
	})

	t.Run("ActiveLawCount counts strictly positive fire rates", func(t *testing.T) {
		t.Parallel()
		c := &coverage.ComponentCoverage{
			FireRate: map[string]float64{"a": 0.5, "b": 0.0, "c": 1.0},
		}
		testkit.Equal(t, c.ActiveLawCount(), 2, "two laws above zero")
	})

	t.Run("WeakLaws filters by explicit threshold", func(t *testing.T) {
		t.Parallel()
		c := &coverage.ComponentCoverage{
			FireRate: map[string]float64{
				"AUTO-RARE": 0.01,
				"AUTO-WEAK": 0.04,
				"AUTO-OK":   0.5,
				"AUTO-LOUD": 1.0,
			},
		}
		testkit.Equal(t, c.WeakLaws(0.05), []string{"AUTO-RARE", "AUTO-WEAK"}, "rates < 0.05")
	})

	t.Run("WeakLaws default threshold is 0.05 when threshold <= 0", func(t *testing.T) {
		t.Parallel()
		c := &coverage.ComponentCoverage{
			FireRate: map[string]float64{
				"AUTO-RARE": 0.04,
				"AUTO-OK":   0.5,
			},
		}
		testkit.Equal(t, c.WeakLaws(0), []string{"AUTO-RARE"}, "rate 0.04 < default 0.05")
		testkit.Equal(t, c.WeakLaws(-1), []string{"AUTO-RARE"}, "negative falls back to default")
	})
}
