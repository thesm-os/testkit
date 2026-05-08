// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package shape_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/generator/shape"
)

func TestWriterDetector(t *testing.T) {
	t.Parallel()
	det := detectorByName(t, "Writer")

	t.Run("matches func(ctx, V) error", func(t *testing.T) {
		t.Parallel()
		sig := buildSig(t, `package p
import "context"
type I interface { F(ctx context.Context, v int) error }
`)
		info, ok := det.Detect(sig)
		testkit.True(t, ok, "Detect")
		testkit.Equal(t, info.Shape, shape.Writer, "Shape")
	})

	t.Run("matches no-ctx form", func(t *testing.T) {
		t.Parallel()
		sig := buildSig(t, `package p
type I interface { F(v int) error }
`)
		_, ok := det.Detect(sig)
		testkit.True(t, ok, "no-ctx Writer")
	})

	t.Run("rejects (V, error) — Reader-shape", func(t *testing.T) {
		t.Parallel()
		sig := buildSig(t, `package p
type I interface { F(v int) (string, error) }
`)
		_, ok := det.Detect(sig)
		testkit.False(t, ok, "non-error result disqualifies (Reader claims it)")
	})

	t.Run("rejects when missing error", func(t *testing.T) {
		t.Parallel()
		sig := buildSig(t, `package p
type I interface { F(v int) }
`)
		_, ok := det.Detect(sig)
		testkit.False(t, ok, "missing error guard")
	})

	t.Run("rejects 0 non-ctx params", func(t *testing.T) {
		t.Parallel()
		sig := buildSig(t, `package p
import "context"
type I interface { F(ctx context.Context) error }
`)
		_, ok := det.Detect(sig)
		testkit.False(t, ok, "Lifecycle-shape rejected")
	})

	t.Run("rejects 2 non-ctx params", func(t *testing.T) {
		t.Parallel()
		sig := buildSig(t, `package p
type I interface { F(a, b string) error }
`)
		_, ok := det.Detect(sig)
		testkit.False(t, ok, "CompositeWriter-shape rejected")
	})

	t.Run("rejects variadic signatures", func(t *testing.T) {
		t.Parallel()
		sig := buildSig(t, `package p
type I interface { F(vs ...int) error }
`)
		_, ok := det.Detect(sig)
		testkit.False(t, ok, "variadic guard")
	})

	t.Run("matches generic signature", func(t *testing.T) {
		t.Parallel()
		sig := buildSig(t, `package p
import "context"
type I[V any] interface { F(ctx context.Context, v V) error }
`)
		info, ok := det.Detect(sig)
		testkit.True(t, ok, "Detect")
		testkit.Equal(t, info.Shape, shape.Writer, "Shape")
	})
}
