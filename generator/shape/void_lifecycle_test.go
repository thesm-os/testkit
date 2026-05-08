// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package shape_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/generator/shape"
)

func TestVoidLifecycleDetector(t *testing.T) {
	t.Parallel()
	det := detectorByName(t, "VoidLifecycle")

	t.Run("matches func() — no ctx, no params, no return", func(t *testing.T) {
		t.Parallel()
		sig := buildSig(t, `package p
type I interface { F() }
`)
		info, ok := det.Detect(sig)
		testkit.True(t, ok, "Detect")
		testkit.Equal(t, info.Shape, shape.VoidLifecycle, "Shape")
	})

	t.Run("matches func(ctx) — ctx-only void", func(t *testing.T) {
		t.Parallel()
		sig := buildSig(t, `package p
import "context"
type I interface { F(ctx context.Context) }
`)
		_, ok := det.Detect(sig)
		testkit.True(t, ok, "ctx + void")
	})

	t.Run("rejects when non-ctx params present", func(t *testing.T) {
		t.Parallel()
		sig := buildSig(t, `package p
type I interface { F(v int) }
`)
		_, ok := det.Detect(sig)
		testkit.False(t, ok, "non-ctx param disqualifies")
	})

	t.Run("rejects when results are present", func(t *testing.T) {
		t.Parallel()
		sig := buildSig(t, `package p
type I interface { F() error }
`)
		_, ok := det.Detect(sig)
		testkit.False(t, ok, "any result disqualifies")
	})

	t.Run("rejects variadic signatures", func(t *testing.T) {
		t.Parallel()
		sig := buildSig(t, `package p
type I interface { F(ks ...string) }
`)
		_, ok := det.Detect(sig)
		testkit.False(t, ok, "variadic guard")
	})
}
