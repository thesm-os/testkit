// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package shape_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/generator/shape"
)

func TestMultiAggregatorDetector(t *testing.T) {
	t.Parallel()
	det := detectorByName(t, "MultiAggregator")

	t.Run("matches func(ctx) (V1, V2, error)", func(t *testing.T) {
		t.Parallel()
		sig := buildSig(t, `package p
import "context"
type I interface { F(ctx context.Context) (int, int, error) }
`)
		info, ok := det.Detect(sig)
		testkit.True(t, ok, "Detect")
		testkit.Equal(t, info.Shape, shape.MultiAggregator, "Shape")
	})

	t.Run("matches no-ctx form when error is present", func(t *testing.T) {
		t.Parallel()
		sig := buildSig(t, `package p
type I interface { F() (int, string, error) }
`)
		_, ok := det.Detect(sig)
		testkit.True(t, ok, "error qualifies w/o ctx")
	})

	t.Run("rejects when neither ctx nor error present", func(t *testing.T) {
		t.Parallel()
		sig := buildSig(t, `package p
type I interface { F() (int, int) }
`)
		_, ok := det.Detect(sig)
		testkit.False(t, ok, "needs ctx OR error")
	})

	t.Run("rejects when non-ctx params are present", func(t *testing.T) {
		t.Parallel()
		sig := buildSig(t, `package p
import "context"
type I interface { F(ctx context.Context, k string) (int, int, error) }
`)
		_, ok := det.Detect(sig)
		testkit.False(t, ok, "MultiReader-shape rejected")
	})

	t.Run("rejects 1 non-error result", func(t *testing.T) {
		t.Parallel()
		sig := buildSig(t, `package p
import "context"
type I interface { F(ctx context.Context) (int, error) }
`)
		_, ok := det.Detect(sig)
		testkit.False(t, ok, "Aggregator-shape rejected")
	})

	t.Run("rejects variadic signatures", func(t *testing.T) {
		t.Parallel()
		sig := buildSig(t, `package p
import "context"
type I interface { F(ctx context.Context, ks ...string) (int, int, error) }
`)
		_, ok := det.Detect(sig)
		testkit.False(t, ok, "variadic guard")
	})

	t.Run("matches generic signature", func(t *testing.T) {
		t.Parallel()
		sig := buildSig(t, `package p
import "context"
type I[V1, V2 any] interface { F(ctx context.Context) (V1, V2, error) }
`)
		info, ok := det.Detect(sig)
		testkit.True(t, ok, "Detect")
		testkit.Equal(t, info.Shape, shape.MultiAggregator, "Shape")
	})
}
