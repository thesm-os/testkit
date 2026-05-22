// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package mutation_test

import (
	"testing"

	"pgregory.net/rapid"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/model/mutation"
)

func TestReturnWrongValue(t *testing.T) {
	t.Parallel()

	t.Run("Name returns the stable identifier", func(t *testing.T) {
		t.Parallel()
		testkit.Equal(t, mutation.ReturnWrongValue[string]{}.Name(), "ReturnWrongValue", "name")
	})

	t.Run("rate 1.0 always retargets", func(t *testing.T) {
		t.Parallel()
		op := mutation.ReturnWrongValue[string]{Rate: 1.0, Alt: rapid.Just("alt")}
		rapid.Check(t, func(rt *rapid.T) {
			alt, ok := op.Retarget(rt)
			if !ok {
				rt.Fatal("rate=1 must always retarget")
			}
			if alt != "alt" {
				rt.Fatalf("alt key wrong: got %q", alt)
			}
		})
	})

	t.Run("rate 0.0 never retargets", func(t *testing.T) {
		t.Parallel()
		op := mutation.ReturnWrongValue[string]{Rate: 0.0, Alt: rapid.Just("alt")}
		rapid.Check(t, func(rt *rapid.T) {
			_, ok := op.Retarget(rt)
			if ok {
				rt.Fatal("rate=0 must never retarget")
			}
		})
	})
}
