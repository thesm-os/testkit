// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package mutation_test

import (
	"testing"

	"pgregory.net/rapid"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/model/mutation"
)

func TestReorderConcurrent(t *testing.T) {
	t.Parallel()

	t.Run("Name returns the stable identifier", func(t *testing.T) {
		t.Parallel()
		testkit.Equal(t, mutation.ReorderConcurrent{}.Name(), "ReorderConcurrent", "name")
	})

	t.Run("rate 1.0 always yields", func(t *testing.T) {
		t.Parallel()
		op := mutation.ReorderConcurrent{Rate: 1.0}
		rapid.Check(t, func(rt *rapid.T) {
			if !op.MaybeYield(rt) {
				rt.Fatal("rate=1 must always yield")
			}
		})
	})

	t.Run("rate 0.0 never yields", func(t *testing.T) {
		t.Parallel()
		op := mutation.ReorderConcurrent{Rate: 0.0}
		rapid.Check(t, func(rt *rapid.T) {
			if op.MaybeYield(rt) {
				rt.Fatal("rate=0 must never yield")
			}
		})
	})
}
