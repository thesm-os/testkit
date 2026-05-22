// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package shrinker_test

import (
	"strconv"
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/model/shrinker"
)

func TestCausalHullIndices(t *testing.T) {
	t.Parallel()

	t.Run("invalid failedAt returns nil", func(t *testing.T) {
		t.Parallel()
		steps := []shrinker.Step{{Name: "a"}}
		testkit.True(t, shrinker.CausalHullIndices(steps, -1) == nil, "negative")
		testkit.True(t, shrinker.CausalHullIndices(steps, 99) == nil, "above range")
	})

	t.Run("isolated failing step returns just itself", func(t *testing.T) {
		t.Parallel()
		steps := []shrinker.Step{
			{Name: "a", Writes: []string{"k1"}},
			{Name: "b", Writes: []string{"k2"}},
			{Name: "c", Reads: []string{"k3"}}, // reads nothing prior writes
		}
		got := shrinker.CausalHullIndices(steps, 2)
		testkit.Equal(t, got, []int{2}, "no prior step satisfies the read")
	})

	t.Run("single dependency", func(t *testing.T) {
		t.Parallel()
		steps := []shrinker.Step{
			{Name: "Put(k)", Writes: []string{"k"}},
			{Name: "noise", Writes: []string{"unrelated"}},
			{Name: "Get(k)", Reads: []string{"k"}},
		}
		got := shrinker.CausalHullIndices(steps, 2)
		testkit.Equal(t, got, []int{0, 2}, "drops noise")
	})

	t.Run("transitive dependency chain", func(t *testing.T) {
		t.Parallel()
		// Write k → Write j (reads k) → Read j
		steps := []shrinker.Step{
			{Name: "putK", Writes: []string{"k"}},
			{Name: "putJ", Reads: []string{"k"}, Writes: []string{"j"}},
			{Name: "noise", Writes: []string{"q"}},
			{Name: "getJ", Reads: []string{"j"}},
		}
		got := shrinker.CausalHullIndices(steps, 3)
		testkit.Equal(t, got, []int{0, 1, 3}, "drops noise but keeps the chain")
	})

	t.Run("only most-recent write per name flows back", func(t *testing.T) {
		t.Parallel()
		// Two prior Put(k); only the second should be in the hull.
		steps := []shrinker.Step{
			{Name: "putK1", Writes: []string{"k"}},
			{Name: "putK2", Writes: []string{"k"}},
			{Name: "getK", Reads: []string{"k"}},
		}
		got := shrinker.CausalHullIndices(steps, 2)
		testkit.Equal(t, got, []int{1, 2}, "older write shadowed")
	})

	t.Run("50-action fixture reduces to its causal hull", func(t *testing.T) {
		t.Parallel()
		// Build 50 steps: 3 form a real causal chain to the
		// failure; the rest are independent noise.
		steps := make([]shrinker.Step, 0, 50)
		// First three: putK → putJ(reads k) → noise (writes q) → ...
		steps = append(steps,
			shrinker.Step{Name: "putK", Writes: []string{"k"}},
			shrinker.Step{Name: "putJ", Reads: []string{"k"}, Writes: []string{"j"}},
		)
		for i := range 47 {
			steps = append(steps, shrinker.Step{
				Name:   "noise-" + strconv.Itoa(i),
				Writes: []string{"noise-" + strconv.Itoa(i)},
			})
		}
		steps = append(steps, shrinker.Step{Name: "getJ", Reads: []string{"j"}})

		got := shrinker.CausalHullIndices(steps, len(steps)-1)
		testkit.Equal(t, got, []int{0, 1, 49}, "50-action sequence reduces to 3")
	})
}

func TestShrinkVerified(t *testing.T) {
	t.Parallel()

	steps := []shrinker.Step{
		{Name: "putK", Writes: []string{"k"}},
		{Name: "noise"},
		{Name: "getK", Reads: []string{"k"}},
	}

	t.Run("verifier accepts → returns the hull", func(t *testing.T) {
		t.Parallel()
		got := shrinker.ShrinkVerified(steps, 2, func(_ []shrinker.Step) bool { return true })
		testkit.Equal(t, len(got), 2, "shrunk")
		testkit.Equal(t, got[0].Name, "putK", "hull start")
		testkit.Equal(t, got[1].Name, "getK", "hull end")
	})

	t.Run("verifier rejects → returns the original sequence", func(t *testing.T) {
		t.Parallel()
		got := shrinker.ShrinkVerified(steps, 2, func(_ []shrinker.Step) bool { return false })
		testkit.Equal(t, len(got), 3, "fell back to original")
	})

	t.Run("empty hull (invalid failedAt) returns original", func(t *testing.T) {
		t.Parallel()
		got := shrinker.ShrinkVerified(steps, -1, func(_ []shrinker.Step) bool { return true })
		testkit.Equal(t, len(got), 3, "invalid failedAt leaves input untouched")
	})
}

func TestCausalHull(t *testing.T) {
	t.Parallel()

	t.Run("returns steps in original order", func(t *testing.T) {
		t.Parallel()
		steps := []shrinker.Step{
			{Name: "put", Writes: []string{"k"}},
			{Name: "noise"},
			{Name: "get", Reads: []string{"k"}},
		}
		got := shrinker.CausalHull(steps, 2)
		testkit.Equal(t, len(got), 2, "two steps")
		testkit.Equal(t, got[0].Name, "put", "put first")
		testkit.Equal(t, got[1].Name, "get", "get second")
	})

	t.Run("empty hull returns nil", func(t *testing.T) {
		t.Parallel()
		testkit.True(t, shrinker.CausalHull(nil, 0) == nil, "empty input")
	})
}
