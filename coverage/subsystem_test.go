// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package coverage_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/coverage"
)

func TestSubsystemCoverageAccessors(t *testing.T) {
	t.Parallel()

	t.Run("nil receiver returns nil/zero across the surface", func(t *testing.T) {
		t.Parallel()
		var s *coverage.SubsystemCoverage
		testkit.Equal(t, len(s.InvariantIDs()), 0, "nil InvariantIDs")
		testkit.Equal(t, len(s.REQIDs()), 0, "nil REQIDs")
		testkit.Equal(t, s.ActiveInvariantCount(), 0, "nil ActiveInvariantCount")
	})

	t.Run("InvariantIDs returns sorted IDs", func(t *testing.T) {
		t.Parallel()
		s := &coverage.SubsystemCoverage{
			InvariantFireRate: map[string]float64{
				"INV-Z": 0.5,
				"INV-A": 1.0,
			},
		}
		testkit.Equal(t, s.InvariantIDs(), []string{"INV-A", "INV-Z"}, "sorted")
	})

	t.Run("REQIDs returns sorted IDs", func(t *testing.T) {
		t.Parallel()
		s := &coverage.SubsystemCoverage{
			SubsystemREQs: map[string][]string{
				"REQ-Z": {"INV-X"},
				"REQ-A": {"INV-Y"},
			},
		}
		testkit.Equal(t, s.REQIDs(), []string{"REQ-A", "REQ-Z"}, "sorted")
	})

	t.Run("ActiveInvariantCount counts strictly positive fire rates", func(t *testing.T) {
		t.Parallel()
		s := &coverage.SubsystemCoverage{
			InvariantFireRate: map[string]float64{"a": 0, "b": 0.5, "c": 1.0},
		}
		testkit.Equal(t, s.ActiveInvariantCount(), 2, "two above zero")
	})
}
