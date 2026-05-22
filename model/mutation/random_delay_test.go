// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package mutation_test

import (
	"testing"
	"time"

	"pgregory.net/rapid"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/model/mutation"
)

func TestRandomDelay(t *testing.T) {
	t.Parallel()

	t.Run("Name returns the stable identifier", func(t *testing.T) {
		t.Parallel()
		testkit.Equal(t, mutation.RandomDelay{}.Name(), "RandomDelay", "name")
	})

	t.Run("delay falls within [Min, Max]", func(t *testing.T) {
		t.Parallel()
		op := mutation.RandomDelay{Min: 10 * time.Microsecond, Max: 100 * time.Microsecond}
		rapid.Check(t, func(rt *rapid.T) {
			d := op.Delay(rt)
			if d < op.Min || d > op.Max {
				rt.Fatalf("delay %v outside [%v, %v]", d, op.Min, op.Max)
			}
		})
	})

	t.Run("Min == Max yields exactly Min", func(t *testing.T) {
		t.Parallel()
		op := mutation.RandomDelay{Min: 50 * time.Microsecond, Max: 50 * time.Microsecond}
		rapid.Check(t, func(rt *rapid.T) {
			d := op.Delay(rt)
			if d != op.Min {
				rt.Fatalf("expected %v, got %v", op.Min, d)
			}
		})
	})

	t.Run("zero / inverted range yields zero", func(t *testing.T) {
		t.Parallel()
		zero := mutation.RandomDelay{}
		inverted := mutation.RandomDelay{Min: 100, Max: 10}
		rapid.Check(t, func(rt *rapid.T) {
			if zero.Delay(rt) != 0 {
				rt.Fatal("zero range must yield 0")
			}
			if inverted.Delay(rt) != 0 {
				rt.Fatal("inverted range must yield 0")
			}
		})
	})
}
