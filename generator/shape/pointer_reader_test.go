// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package shape_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/generator/shape"
)

func TestPointerReaderDetector(t *testing.T) {
	t.Parallel()
	det := detectorByName(t, "PointerReader")

	t.Run("matches func(ctx, K) *V", func(t *testing.T) {
		t.Parallel()
		sig := buildSig(t, `package p
import "context"
type Item struct{}
type I interface { F(ctx context.Context, k string) *Item }
`)
		info, ok := det.Detect(sig)
		testkit.True(t, ok, "Detect")
		testkit.Equal(t, info.Shape, shape.PointerReader, "Shape")
		testkit.Equal(t, info.ValType, "Item", "ValType is the pointee")
	})

	t.Run("matches no-ctx form", func(t *testing.T) {
		t.Parallel()
		sig := buildSig(t, `package p
type Item struct{}
type I interface { F(k string) *Item }
`)
		_, ok := det.Detect(sig)
		testkit.True(t, ok, "no-ctx variant")
	})

	t.Run("rejects non-pointer result", func(t *testing.T) {
		t.Parallel()
		sig := buildSig(t, `package p
type I interface { F(k string) int }
`)
		_, ok := det.Detect(sig)
		testkit.False(t, ok, "non-pointer V")
	})

	t.Run("rejects when error result is present", func(t *testing.T) {
		t.Parallel()
		sig := buildSig(t, `package p
type Item struct{}
type I interface { F(k string) (*Item, error) }
`)
		_, ok := det.Detect(sig)
		testkit.False(t, ok, "Reader-shape rejected")
	})

	t.Run("rejects 0 non-ctx params", func(t *testing.T) {
		t.Parallel()
		sig := buildSig(t, `package p
type Item struct{}
type I interface { F() *Item }
`)
		_, ok := det.Detect(sig)
		testkit.False(t, ok, "needs key param")
	})

	t.Run("rejects variadic signatures", func(t *testing.T) {
		t.Parallel()
		sig := buildSig(t, `package p
type Item struct{}
type I interface { F(ks ...string) *Item }
`)
		_, ok := det.Detect(sig)
		testkit.False(t, ok, "variadic guard")
	})

	t.Run("matches generic signature with pointer-to-typeparam", func(t *testing.T) {
		t.Parallel()
		sig := buildSig(t, `package p
type I[K comparable, V any] interface { F(k K) *V }
`)
		info, ok := det.Detect(sig)
		testkit.True(t, ok, "Detect")
		testkit.Equal(t, info.Shape, shape.PointerReader, "Shape")
	})
}
