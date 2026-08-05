// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package mutation_test

import (
	"testing"

	"pgregory.net/rapid"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/engine/model/mutation"
)

func TestDropWrites(t *testing.T) {
	t.Parallel()

	t.Run("Name returns the stable identifier", func(t *testing.T) {
		t.Parallel()
		testkit.Equal(t, mutation.DropWrites{}.Name(), "DropWrites", "name")
	})

	t.Run("rate 1.0 always drops", func(t *testing.T) {
		t.Parallel()
		op := mutation.DropWrites{Rate: 1.0}
		rapid.Check(t, func(rt *rapid.T) {
			if !op.ShouldDrop(rt) {
				rt.Fatal("rate=1 must always drop")
			}
		})
	})

	t.Run("rate 0.0 never drops", func(t *testing.T) {
		t.Parallel()
		op := mutation.DropWrites{Rate: 0.0}
		rapid.Check(t, func(rt *rapid.T) {
			if op.ShouldDrop(rt) {
				rt.Fatal("rate=0 must never drop")
			}
		})
	})

	t.Run("rate 0.5 produces both outcomes across iterations", func(t *testing.T) {
		t.Parallel()
		op := mutation.DropWrites{Rate: 0.5}
		var sawDrop, sawKeep bool
		rapid.Check(t, func(rt *rapid.T) {
			if op.ShouldDrop(rt) {
				sawDrop = true
			} else {
				sawKeep = true
			}
		})
		if !sawDrop {
			t.Fatal("rate=0.5 never dropped across 100 iterations")
		}
		if !sawKeep {
			t.Fatal("rate=0.5 always dropped across 100 iterations")
		}
	})
}
