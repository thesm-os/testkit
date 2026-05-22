// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package mutation_test

import (
	"testing"

	"pgregory.net/rapid"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/model/mutation"
)

func TestLossyStream(t *testing.T) {
	t.Parallel()

	t.Run("Name returns the stable identifier", func(t *testing.T) {
		t.Parallel()
		testkit.Equal(t, mutation.LossyStream{}.Name(), "LossyStream", "name")
	})

	t.Run("rate 1.0 always drops", func(t *testing.T) {
		t.Parallel()
		op := mutation.LossyStream{Rate: 1.0}
		rapid.Check(t, func(rt *rapid.T) {
			if !op.ShouldDrop(rt) {
				rt.Fatal("rate=1 must always drop")
			}
		})
	})

	t.Run("rate 0.0 never drops", func(t *testing.T) {
		t.Parallel()
		op := mutation.LossyStream{Rate: 0.0}
		rapid.Check(t, func(rt *rapid.T) {
			if op.ShouldDrop(rt) {
				rt.Fatal("rate=0 must never drop")
			}
		})
	})
}

func TestOperatorInterface(t *testing.T) {
	t.Parallel()

	t.Run("every shipped operator implements the Operator interface", func(t *testing.T) {
		t.Parallel()
		ops := []mutation.Operator{
			mutation.DropWrites{},
			mutation.ReturnWrongValue[string]{},
			mutation.MissDeletes{},
			mutation.RandomDelay{},
			mutation.ReorderConcurrent{},
			mutation.OffByOneIndex{},
			mutation.DuplicateAppends{},
			mutation.LossyStream{},
		}
		seen := make(map[string]bool, len(ops))
		for _, o := range ops {
			if n := o.Name(); n == "" {
				t.Fatalf("operator name empty: %T", o)
			} else if seen[n] {
				t.Fatalf("duplicate operator name: %q", n)
			} else {
				seen[n] = true
			}
		}
	})
}
