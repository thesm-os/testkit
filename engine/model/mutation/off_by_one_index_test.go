// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package mutation_test

import (
	"testing"

	"pgregory.net/rapid"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/engine/model/mutation"
)

func TestOffByOneIndex(t *testing.T) {
	t.Parallel()

	t.Run("Name returns the stable identifier", func(t *testing.T) {
		t.Parallel()
		testkit.Equal(t, mutation.OffByOneIndex{}.Name(), "OffByOneIndex", "name")
	})

	t.Run("rate 1.0 always perturbs by exactly 1 in either direction", func(t *testing.T) {
		t.Parallel()
		op := mutation.OffByOneIndex{Rate: 1.0}
		rapid.Check(t, func(rt *rapid.T) {
			got := op.Mutate(rt, 100)
			if got != 99 && got != 101 {
				rt.Fatalf("rate=1 must yield 99 or 101 from 100, got %d", got)
			}
		})
	})

	t.Run("rate 0.0 never perturbs", func(t *testing.T) {
		t.Parallel()
		op := mutation.OffByOneIndex{Rate: 0.0}
		rapid.Check(t, func(rt *rapid.T) {
			if op.Mutate(rt, 42) != 42 {
				rt.Fatal("rate=0 must pass n through")
			}
		})
	})
}
