// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package shape_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/generator/shape"
)

func TestBatchReaderDetector(t *testing.T) {
	t.Parallel()
	det := detectorByName(t, "BatchReader")

	t.Run("matches func(ctx, ...K) ([]V, error)", func(t *testing.T) {
		t.Parallel()
		sig := buildSig(t, `package p
import "context"
type I interface { F(ctx context.Context, ids ...string) ([]int, error) }
`)
		info, ok := det.Detect(sig)
		testkit.True(t, ok, "Detect")
		testkit.Equal(t, info.Shape, shape.BatchReader, "Shape")
		testkit.Equal(t, info.KeyType, "string", "KeyType from variadic elem")
		testkit.Equal(t, info.ValType, "int", "ValType from slice elem")
	})

	t.Run("matches no-ctx form", func(t *testing.T) {
		t.Parallel()
		sig := buildSig(t, `package p
type I interface { F(ids ...string) ([]int, error) }
`)
		_, ok := det.Detect(sig)
		testkit.True(t, ok, "no-ctx variadic")
	})

	t.Run("rejects non-variadic signatures", func(t *testing.T) {
		t.Parallel()
		sig := buildSig(t, `package p
import "context"
type I interface { F(ctx context.Context, ids []string) ([]int, error) }
`)
		_, ok := det.Detect(sig)
		testkit.False(t, ok, "slice param not variadic")
	})

	t.Run("rejects when non-variadic params are present", func(t *testing.T) {
		t.Parallel()
		sig := buildSig(t, `package p
import "context"
type I interface { F(ctx context.Context, ns string, ids ...string) ([]int, error) }
`)
		_, ok := det.Detect(sig)
		testkit.False(t, ok, "extra non-ctx param disqualifies")
	})

	t.Run("rejects when no error result", func(t *testing.T) {
		t.Parallel()
		sig := buildSig(t, `package p
import "context"
type I interface { F(ctx context.Context, ids ...string) []int }
`)
		_, ok := det.Detect(sig)
		testkit.False(t, ok, "missing error guard")
	})

	t.Run("rejects when result is not a slice", func(t *testing.T) {
		t.Parallel()
		sig := buildSig(t, `package p
import "context"
type I interface { F(ctx context.Context, ids ...string) (int, error) }
`)
		_, ok := det.Detect(sig)
		testkit.False(t, ok, "scalar result rejected")
	})

	t.Run("matches generic signature", func(t *testing.T) {
		t.Parallel()
		sig := buildSig(t, `package p
type I[K comparable, V any] interface { F(ks ...K) ([]V, error) }
`)
		info, ok := det.Detect(sig)
		testkit.True(t, ok, "Detect")
		testkit.Equal(t, info.Shape, shape.BatchReader, "Shape")
	})
}
