// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package refmachine_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/model/refmachine"
)

func TestFoldMachine(t *testing.T) {
	t.Parallel()

	t.Run("Apply accumulates via fold", func(t *testing.T) {
		t.Parallel()
		m := refmachine.NewFoldMachine[int, int](0, func(s, p int) int { return s + p })
		_ = m.Apply(t.Context(), 1)
		_ = m.Apply(t.Context(), 2)
		_ = m.Apply(t.Context(), 3)
		s, _ := m.State(t.Context())
		testkit.Equal(t, s, 6, "sum")
	})

	t.Run("initial state visible without Apply", func(t *testing.T) {
		t.Parallel()
		m := refmachine.NewFoldMachine[int, string]("seed", func(s string, _ int) string { return s })
		s, _ := m.State(t.Context())
		testkit.Equal(t, s, "seed", "initial preserved")
	})
}
