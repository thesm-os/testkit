// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package shape_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/generator/shape"
)

func TestCompositeWriterDetector(t *testing.T) {
	t.Parallel()
	det := detectorByName(t, "CompositeWriter")

	t.Run("matches func(ctx, K1, V) error", func(t *testing.T) {
		t.Parallel()
		sig := buildSig(t, `package p
import "context"
type I interface { F(ctx context.Context, tag string, v int) error }
`)
		info, ok := det.Detect(sig)
		testkit.True(t, ok, "Detect")
		testkit.Equal(t, info.Shape, shape.CompositeWriter, "Shape")
	})

	t.Run("matches no-ctx form", func(t *testing.T) {
		t.Parallel()
		sig := buildSig(t, `package p
type I interface { F(tag string, v int) error }
`)
		_, ok := det.Detect(sig)
		testkit.True(t, ok, "no-ctx 2-param Writer")
	})

	t.Run("rejects with 1 non-ctx param", func(t *testing.T) {
		t.Parallel()
		sig := buildSig(t, `package p
import "context"
type I interface { F(ctx context.Context, k string) error }
`)
		_, ok := det.Detect(sig)
		testkit.False(t, ok, "Writer-shape, not CompositeWriter")
	})

	t.Run("rejects with 3+ non-ctx params", func(t *testing.T) {
		t.Parallel()
		sig := buildSig(t, `package p
import "context"
type I interface { F(ctx context.Context, a, b, c string) error }
`)
		_, ok := det.Detect(sig)
		testkit.False(t, ok, "MultiArgWriter-shape rejected")
	})

	t.Run("rejects without error return", func(t *testing.T) {
		t.Parallel()
		sig := buildSig(t, `package p
type I interface { F(a string, b int) }
`)
		_, ok := det.Detect(sig)
		testkit.False(t, ok, "missing error guard")
	})

	t.Run("rejects when non-error results present", func(t *testing.T) {
		t.Parallel()
		sig := buildSig(t, `package p
type I interface { F(a string, b int) (int, error) }
`)
		_, ok := det.Detect(sig)
		testkit.False(t, ok, "extra non-error result rejected")
	})

	t.Run("rejects variadic signatures", func(t *testing.T) {
		t.Parallel()
		sig := buildSig(t, `package p
type I interface { F(a string, bs ...int) error }
`)
		_, ok := det.Detect(sig)
		testkit.False(t, ok, "variadic guard")
	})

	t.Run("matches generic signature", func(t *testing.T) {
		t.Parallel()
		sig := buildSig(t, `package p
import "context"
type I[K comparable, V any] interface { F(ctx context.Context, k K, v V) error }
`)
		info, ok := det.Detect(sig)
		testkit.True(t, ok, "Detect")
		testkit.Equal(t, info.Shape, shape.CompositeWriter, "Shape")
	})
}
