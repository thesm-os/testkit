// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package shape_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/generator/shape"
)

func TestMultiReaderDetector(t *testing.T) {
	t.Parallel()
	det := detectorByName(t, "MultiReader")

	t.Run("matches func(ctx, K) (V1, V2, error)", func(t *testing.T) {
		t.Parallel()
		sig := buildSig(t, `package p
import "context"
type I interface { F(ctx context.Context, k string) (int, string, error) }
`)
		info, ok := det.Detect(sig)
		testkit.True(t, ok, "Detect")
		testkit.Equal(t, info.Shape, shape.MultiReader, "Shape")
	})

	t.Run("matches no-ctx form", func(t *testing.T) {
		t.Parallel()
		sig := buildSig(t, `package p
type I interface { F(k string) (int, string, error) }
`)
		_, ok := det.Detect(sig)
		testkit.True(t, ok, "no-ctx variant")
	})

	t.Run("rejects 0 non-ctx params", func(t *testing.T) {
		t.Parallel()
		sig := buildSig(t, `package p
import "context"
type I interface { F(ctx context.Context) (int, string, error) }
`)
		_, ok := det.Detect(sig)
		testkit.False(t, ok, "needs key param")
	})

	t.Run("rejects without error return", func(t *testing.T) {
		t.Parallel()
		sig := buildSig(t, `package p
type I interface { F(k string) (int, string, bool) }
`)
		_, ok := det.Detect(sig)
		testkit.False(t, ok, "missing error guard")
	})

	t.Run("rejects when not exactly 2 non-error results", func(t *testing.T) {
		t.Parallel()
		sig := buildSig(t, `package p
type I interface { F(k string) (int, error) }
`)
		_, ok := det.Detect(sig)
		testkit.False(t, ok, "single non-error → Reader, not MultiReader")
	})

	t.Run("rejects variadic signatures", func(t *testing.T) {
		t.Parallel()
		sig := buildSig(t, `package p
type I interface { F(ks ...string) (int, string, error) }
`)
		_, ok := det.Detect(sig)
		testkit.False(t, ok, "variadic guard")
	})

	t.Run("matches generic signature", func(t *testing.T) {
		t.Parallel()
		sig := buildSig(t, `package p
type I[K comparable, V1 any, V2 any] interface { F(k K) (V1, V2, error) }
`)
		info, ok := det.Detect(sig)
		testkit.True(t, ok, "Detect")
		testkit.Equal(t, info.Shape, shape.MultiReader, "Shape")
	})
}
