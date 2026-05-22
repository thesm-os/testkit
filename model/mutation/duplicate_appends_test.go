// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package mutation_test

import (
	"testing"

	"pgregory.net/rapid"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/model/mutation"
)

func TestDuplicateAppends(t *testing.T) {
	t.Parallel()

	t.Run("Name returns the stable identifier", func(t *testing.T) {
		t.Parallel()
		testkit.Equal(t, mutation.DuplicateAppends{}.Name(), "DuplicateAppends", "name")
	})

	t.Run("rate 1.0 always duplicates", func(t *testing.T) {
		t.Parallel()
		op := mutation.DuplicateAppends{Rate: 1.0}
		rapid.Check(t, func(rt *rapid.T) {
			if !op.ShouldDuplicate(rt) {
				rt.Fatal("rate=1 must always duplicate")
			}
		})
	})

	t.Run("rate 0.0 never duplicates", func(t *testing.T) {
		t.Parallel()
		op := mutation.DuplicateAppends{Rate: 0.0}
		rapid.Check(t, func(rt *rapid.T) {
			if op.ShouldDuplicate(rt) {
				rt.Fatal("rate=0 must never duplicate")
			}
		})
	})
}
