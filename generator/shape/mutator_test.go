// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package shape_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/generator/directive"
	"go.thesmos.sh/testkit/generator/shape"
)

func TestMutatorDetector(t *testing.T) {
	t.Parallel()
	det := detectorByName(t, "Mutator")

	t.Run("auto-detects func(ctx, V) without directive", func(t *testing.T) {
		t.Parallel()
		sig := buildSig(t, `package p
import "context"
type I interface { F(ctx context.Context, v int) }
`)
		info, ok := det.Detect(sig)
		testkit.True(t, ok, "Detect")
		testkit.Equal(t, info.Shape, shape.Mutator, "Shape")
	})

	t.Run("auto-detects no-ctx form func(V)", func(t *testing.T) {
		t.Parallel()
		sig := buildSig(t, `package p
type I interface { F(v int) }
`)
		_, ok := det.Detect(sig)
		testkit.True(t, ok, "no-ctx Mutator")
	})

	t.Run("//testkit:not-mutator opts out", func(t *testing.T) {
		t.Parallel()
		sig := buildSig(t, `package p
import "context"
type I interface { F(ctx context.Context, v int) }
`, directive.Directive{Name: directive.NotMutator})
		_, ok := det.Detect(sig)
		testkit.False(t, ok, "not-mutator opts out")
	})

	t.Run("//testkit:directive mutator=off opts out", func(t *testing.T) {
		t.Parallel()
		sig := buildSig(t, `package p
import "context"
type I interface { F(ctx context.Context, v int) }
`, directive.Directive{Name: directive.Mutator, Off: true})
		_, ok := det.Detect(sig)
		testkit.False(t, ok, "mutator=off opts out")
	})

	t.Run("rejects when results are present", func(t *testing.T) {
		t.Parallel()
		sig := buildSig(t, `package p
import "context"
type I interface { F(ctx context.Context, v int) error }
`)
		_, ok := det.Detect(sig)
		testkit.False(t, ok, "any return disqualifies (Writer-shape)")
	})

	t.Run("rejects 0 non-ctx params", func(t *testing.T) {
		t.Parallel()
		sig := buildSig(t, `package p
import "context"
type I interface { F(ctx context.Context) }
`)
		_, ok := det.Detect(sig)
		testkit.False(t, ok, "needs at least 1 non-ctx param")
	})

	t.Run("rejects 2 non-ctx params", func(t *testing.T) {
		t.Parallel()
		sig := buildSig(t, `package p
type I interface { F(a, b int) }
`)
		_, ok := det.Detect(sig)
		testkit.False(t, ok, "exactly 1 non-ctx param required")
	})

	t.Run("rejects variadic signatures", func(t *testing.T) {
		t.Parallel()
		sig := buildSig(t, `package p
type I interface { F(vs ...int) }
`)
		_, ok := det.Detect(sig)
		testkit.False(t, ok, "variadic guard")
	})

	t.Run("matches generic signature", func(t *testing.T) {
		t.Parallel()
		sig := buildSig(t, `package p
import "context"
type I[V any] interface { F(ctx context.Context, v V) }
`)
		info, ok := det.Detect(sig)
		testkit.True(t, ok, "Detect")
		testkit.Equal(t, info.Shape, shape.Mutator, "Shape")
	})
}
