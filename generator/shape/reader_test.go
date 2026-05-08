// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package shape_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/generator/shape"
)

func TestReaderDetector(t *testing.T) {
	t.Parallel()
	det := detectorByName(t, "Reader")

	t.Run("matches func(ctx, K) (V, error)", func(t *testing.T) {
		t.Parallel()
		sig := buildSig(t, `package p
import "context"
type I interface { F(ctx context.Context, k string) (int, error) }
`)
		info, ok := det.Detect(sig)
		testkit.True(t, ok, "Detect")
		testkit.Equal(t, info.Shape, shape.Reader, "Shape")
		testkit.Equal(t, info.KeyType, "string", "KeyType")
		testkit.Equal(t, info.ValType, "int", "ValType")
	})

	t.Run("matches no-ctx form (G29)", func(t *testing.T) {
		t.Parallel()
		sig := buildSig(t, `package p
type I interface { F(k string) (int, error) }
`)
		_, ok := det.Detect(sig)
		testkit.True(t, ok, "ctx-optional Reader")
	})

	t.Run("rejects when no error result", func(t *testing.T) {
		t.Parallel()
		sig := buildSig(t, `package p
type I interface { F(k string) int }
`)
		_, ok := det.Detect(sig)
		testkit.False(t, ok, "ReaderNoError-shape rejected")
	})

	t.Run("rejects 0 non-ctx params", func(t *testing.T) {
		t.Parallel()
		sig := buildSig(t, `package p
import "context"
type I interface { F(ctx context.Context) (int, error) }
`)
		_, ok := det.Detect(sig)
		testkit.False(t, ok, "Aggregator-shape rejected")
	})

	t.Run("rejects multi non-error results", func(t *testing.T) {
		t.Parallel()
		sig := buildSig(t, `package p
type I interface { F(k string) (int, string, error) }
`)
		_, ok := det.Detect(sig)
		testkit.False(t, ok, "MultiReader-shape rejected")
	})

	t.Run("rejects variadic signatures", func(t *testing.T) {
		t.Parallel()
		sig := buildSig(t, `package p
type I interface { F(ks ...string) (int, error) }
`)
		_, ok := det.Detect(sig)
		testkit.False(t, ok, "variadic guard")
	})

	t.Run("matches generic signature with type-parameter K and V", func(t *testing.T) {
		t.Parallel()
		sig := buildSig(t, `package p
import "context"
type I[K comparable, V any] interface { F(ctx context.Context, k K) (V, error) }
`)
		info, ok := det.Detect(sig)
		testkit.True(t, ok, "Detect")
		testkit.Equal(t, info.Shape, shape.Reader, "Shape")
		testkit.Equal(t, info.KeyType, "K", "KeyType renders as type-param symbol")
		testkit.Equal(t, info.ValType, "V", "ValType renders as type-param symbol")
	})
}
