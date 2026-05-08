// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package shape_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/generator/shape"
)

func TestAggregatorDetector(t *testing.T) {
	t.Parallel()
	det := detectorByName(t, "Aggregator")

	t.Run("matches func(ctx) (T, error)", func(t *testing.T) {
		t.Parallel()
		sig := buildSig(t, `package p
import "context"
type I interface { F(ctx context.Context) (int, error) }
`)
		info, ok := det.Detect(sig)
		testkit.True(t, ok, "Detect")
		testkit.Equal(t, info.Shape, shape.Aggregator, "Shape")
		testkit.Equal(t, info.ValType, "int", "ValType")
	})

	t.Run("matches no-error form func(ctx) T (G28-extended)", func(t *testing.T) {
		t.Parallel()
		sig := buildSig(t, `package p
import "context"
type I interface { F(ctx context.Context) int }
`)
		_, ok := det.Detect(sig)
		testkit.True(t, ok, "ctx + T (no error)")
	})

	t.Run("matches error-only form func() (T, error)", func(t *testing.T) {
		t.Parallel()
		sig := buildSig(t, `package p
type I interface { F() (int, error) }
`)
		_, ok := det.Detect(sig)
		testkit.True(t, ok, "no ctx but error → still Aggregator")
	})

	t.Run("rejects when neither ctx nor error", func(t *testing.T) {
		t.Parallel()
		sig := buildSig(t, `package p
type I interface { F() int }
`)
		_, ok := det.Detect(sig)
		testkit.False(t, ok, "no ctx, no error → rejected")
	})

	t.Run("rejects variadic signatures", func(t *testing.T) {
		t.Parallel()
		sig := buildSig(t, `package p
import "context"
type I interface { F(ctx context.Context, ks ...string) (int, error) }
`)
		_, ok := det.Detect(sig)
		testkit.False(t, ok, "variadic guard")
	})

	t.Run("rejects when non-ctx params present", func(t *testing.T) {
		t.Parallel()
		sig := buildSig(t, `package p
import "context"
type I interface { F(ctx context.Context, k string) (int, error) }
`)
		_, ok := det.Detect(sig)
		testkit.False(t, ok, "non-ctx param guard")
	})

	t.Run("rejects multi-result returns", func(t *testing.T) {
		t.Parallel()
		sig := buildSig(t, `package p
import "context"
type I interface { F(ctx context.Context) (int, int, error) }
`)
		_, ok := det.Detect(sig)
		testkit.False(t, ok, "multi non-error result rejected")
	})

	t.Run("rejects error-only return", func(t *testing.T) {
		t.Parallel()
		sig := buildSig(t, `package p
import "context"
type I interface { F(ctx context.Context) error }
`)
		_, ok := det.Detect(sig)
		testkit.False(t, ok, "0 non-error results rejected")
	})

	t.Run("matches generic signature", func(t *testing.T) {
		t.Parallel()
		sig := buildSig(t, `package p
import "context"
type I[V any] interface { F(ctx context.Context) (V, error) }
`)
		info, ok := det.Detect(sig)
		testkit.True(t, ok, "Detect")
		testkit.Equal(t, info.Shape, shape.Aggregator, "Shape")
	})
}
