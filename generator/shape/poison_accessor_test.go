// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package shape_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/generator/shape"
)

func TestPoisonAccessorDetector(t *testing.T) {
	t.Parallel()
	det := detectorByName(t, "PoisonAccessor")

	t.Run("matches func() error", func(t *testing.T) {
		t.Parallel()
		sig := buildSig(t, `package p
type I interface { F() error }
`)
		info, ok := det.Detect(sig)
		testkit.True(t, ok, "Detect")
		testkit.Equal(t, info.Shape, shape.PoisonAccessor, "Shape")
	})

	t.Run("rejects when ctx is present", func(t *testing.T) {
		t.Parallel()
		sig := buildSig(t, `package p
import "context"
type I interface { F(ctx context.Context) error }
`)
		_, ok := det.Detect(sig)
		testkit.False(t, ok, "ctx disqualifies")
	})

	t.Run("rejects when params are present", func(t *testing.T) {
		t.Parallel()
		sig := buildSig(t, `package p
type I interface { F(k string) error }
`)
		_, ok := det.Detect(sig)
		testkit.False(t, ok, "non-ctx param disqualifies")
	})

	t.Run("rejects when no error", func(t *testing.T) {
		t.Parallel()
		sig := buildSig(t, `package p
type I interface { F() }
`)
		_, ok := det.Detect(sig)
		testkit.False(t, ok, "missing error guard")
	})

	t.Run("rejects when non-error results present", func(t *testing.T) {
		t.Parallel()
		sig := buildSig(t, `package p
type I interface { F() (int, error) }
`)
		_, ok := det.Detect(sig)
		testkit.False(t, ok, "(T, error) is Aggregator")
	})

	t.Run("rejects variadic signatures", func(t *testing.T) {
		t.Parallel()
		sig := buildSig(t, `package p
type I interface { F(ks ...string) error }
`)
		_, ok := det.Detect(sig)
		testkit.False(t, ok, "variadic guard")
	})
}
