// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package shape_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/generator/shape"
)

func TestPureDetector(t *testing.T) {
	t.Parallel()
	det := detectorByName(t, "Pure")

	t.Run("matches func() T", func(t *testing.T) {
		t.Parallel()
		sig := buildSig(t, `package p
type I interface { F() float64 }
`)
		info, ok := det.Detect(sig)
		testkit.True(t, ok, "Detect")
		testkit.Equal(t, info.Shape, shape.Pure, "Shape")
		testkit.Equal(t, info.ValType, "float64", "ValType")
	})

	t.Run("rejects when ctx is present", func(t *testing.T) {
		t.Parallel()
		sig := buildSig(t, `package p
import "context"
type I interface { F(ctx context.Context) int }
`)
		_, ok := det.Detect(sig)
		testkit.False(t, ok, "ctx disqualifies")
	})

	t.Run("rejects when params are present", func(t *testing.T) {
		t.Parallel()
		sig := buildSig(t, `package p
type I interface { F(k string) int }
`)
		_, ok := det.Detect(sig)
		testkit.False(t, ok, "non-ctx param disqualifies")
	})

	t.Run("rejects when error is present", func(t *testing.T) {
		t.Parallel()
		sig := buildSig(t, `package p
type I interface { F() (int, error) }
`)
		_, ok := det.Detect(sig)
		testkit.False(t, ok, "error rejected")
	})

	t.Run("rejects multi-result", func(t *testing.T) {
		t.Parallel()
		sig := buildSig(t, `package p
type I interface { F() (int, int) }
`)
		_, ok := det.Detect(sig)
		testkit.False(t, ok, "multi-result rejected")
	})

	t.Run("rejects variadic signatures", func(t *testing.T) {
		t.Parallel()
		sig := buildSig(t, `package p
type I interface { F(ks ...string) int }
`)
		_, ok := det.Detect(sig)
		testkit.False(t, ok, "variadic guard")
	})
}
