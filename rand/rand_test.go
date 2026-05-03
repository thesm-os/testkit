// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package rand_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/rand"
)

func TestDefaultRandSource(t *testing.T) {
	t.Parallel()

	t.Run("returns values in [0, 1)", func(t *testing.T) {
		t.Parallel()
		src := rand.DefaultRandSource()
		for range 100 {
			v := src.Float64()
			testkit.True(t, v >= 0 && v < 1, "Float64 must be in [0, 1)")
		}
	})
}

func TestFixedRandSource(t *testing.T) {
	t.Parallel()

	t.Run("always returns configured value", func(t *testing.T) {
		t.Parallel()
		src := rand.FixedRandSource(0.42)
		for range 10 {
			testkit.Equal(t, src.Float64(), 0.42, "must return fixed value")
		}
	})

	t.Run("zero value", func(t *testing.T) {
		t.Parallel()
		src := rand.FixedRandSource(0.0)
		testkit.Equal(t, src.Float64(), 0.0, "must return zero")
	})
}
