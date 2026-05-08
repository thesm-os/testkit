// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package shape_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/generator/shape"
)

func TestReaderNoErrorDetector(t *testing.T) {
	t.Parallel()
	det := detectorByName(t, "ReaderNoError")

	t.Run("matches func(K) V", func(t *testing.T) {
		t.Parallel()
		sig := buildSig(t, `package p
type I interface { F(k string) int }
`)
		info, ok := det.Detect(sig)
		testkit.True(t, ok, "Detect")
		testkit.Equal(t, info.Shape, shape.ReaderNoError, "Shape")
	})

	t.Run("matches func(ctx, K) V", func(t *testing.T) {
		t.Parallel()
		sig := buildSig(t, `package p
import "context"
type I interface { F(ctx context.Context, k string) int }
`)
		_, ok := det.Detect(sig)
		testkit.True(t, ok, "ReaderNoError with ctx")
	})

	t.Run("rejects when error result is present", func(t *testing.T) {
		t.Parallel()
		sig := buildSig(t, `package p
type I interface { F(k string) (int, error) }
`)
		_, ok := det.Detect(sig)
		testkit.False(t, ok, "Reader-shape rejected")
	})

	t.Run("rejects 0 non-ctx params", func(t *testing.T) {
		t.Parallel()
		sig := buildSig(t, `package p
type I interface { F() int }
`)
		_, ok := det.Detect(sig)
		testkit.False(t, ok, "Pure-shape rejected")
	})

	t.Run("rejects multi-result", func(t *testing.T) {
		t.Parallel()
		sig := buildSig(t, `package p
type I interface { F(k string) (int, int) }
`)
		_, ok := det.Detect(sig)
		testkit.False(t, ok, "single non-error result required")
	})

	t.Run("rejects variadic signatures", func(t *testing.T) {
		t.Parallel()
		sig := buildSig(t, `package p
type I interface { F(ks ...string) int }
`)
		_, ok := det.Detect(sig)
		testkit.False(t, ok, "variadic guard")
	})

	t.Run("matches generic signature", func(t *testing.T) {
		t.Parallel()
		sig := buildSig(t, `package p
type I[K comparable, V any] interface { F(k K) V }
`)
		info, ok := det.Detect(sig)
		testkit.True(t, ok, "Detect")
		testkit.Equal(t, info.Shape, shape.ReaderNoError, "Shape")
	})
}
