// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package mutation_test

import (
	"testing"

	"pgregory.net/rapid"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/engine/model/mutation"
)

func TestMissDeletes(t *testing.T) {
	t.Parallel()

	t.Run("Name returns the stable identifier", func(t *testing.T) {
		t.Parallel()
		testkit.Equal(t, mutation.MissDeletes{}.Name(), "MissDeletes", "name")
	})

	t.Run("rate 1.0 always misses", func(t *testing.T) {
		t.Parallel()
		op := mutation.MissDeletes{Rate: 1.0}
		rapid.Check(t, func(rt *rapid.T) {
			if !op.ShouldMiss(rt) {
				rt.Fatal("rate=1 must always miss")
			}
		})
	})

	t.Run("rate 0.0 never misses", func(t *testing.T) {
		t.Parallel()
		op := mutation.MissDeletes{Rate: 0.0}
		rapid.Check(t, func(rt *rapid.T) {
			if op.ShouldMiss(rt) {
				rt.Fatal("rate=0 must never miss")
			}
		})
	})
}
