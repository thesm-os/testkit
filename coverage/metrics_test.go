// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package coverage_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/coverage"
)

func TestBranchCoverageMetricsRatio(t *testing.T) {
	t.Parallel()

	t.Run("zero Total returns zero ratio without dividing", func(t *testing.T) {
		t.Parallel()
		m := coverage.BranchCoverageMetrics{Hit: 5, Total: 0}
		testkit.Equal(t, m.Ratio(), 0.0, "Total=0 sentinel suppresses Hit")
	})

	t.Run("non-zero Total returns Hit/Total", func(t *testing.T) {
		t.Parallel()
		m := coverage.BranchCoverageMetrics{Hit: 3, Total: 4}
		testkit.Equal(t, m.Ratio(), 0.75, "3 of 4 branches")
	})
}
