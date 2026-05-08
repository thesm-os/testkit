// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package shape_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/generator/shape"
)

func TestMultiArgWriterDetector(t *testing.T) {
	t.Parallel()
	det := detectorByName(t, "MultiArgWriter")

	t.Run("matches func(ctx, p1, p2, p3, ...) error with 3+ params", func(t *testing.T) {
		t.Parallel()
		sig := buildSig(t, `package p
import "context"
type I interface {
	F(ctx context.Context, id string, n int, fn func() error) error
}
`)
		info, ok := det.Detect(sig)
		testkit.True(t, ok, "Detect")
		testkit.Equal(t, info.Shape, shape.MultiArgWriter, "Shape")
	})

	t.Run("rejects when ctx is missing", func(t *testing.T) {
		t.Parallel()
		sig := buildSig(t, `package p
type I interface { F(a, b, c string) error }
`)
		_, ok := det.Detect(sig)
		testkit.False(t, ok, "ctx-required guard")
	})

	t.Run("rejects with fewer than 3 non-ctx params", func(t *testing.T) {
		t.Parallel()
		sig := buildSig(t, `package p
import "context"
type I interface { F(ctx context.Context, a, b string) error }
`)
		_, ok := det.Detect(sig)
		testkit.False(t, ok, "≥ 3 non-ctx params required")
	})

	t.Run("rejects without error return", func(t *testing.T) {
		t.Parallel()
		sig := buildSig(t, `package p
import "context"
type I interface { F(ctx context.Context, a, b, c string) }
`)
		_, ok := det.Detect(sig)
		testkit.False(t, ok, "missing error guard")
	})

	t.Run("rejects multi non-error results", func(t *testing.T) {
		t.Parallel()
		sig := buildSig(t, `package p
import "context"
type I interface { F(ctx context.Context, a, b, c string) (int, int, error) }
`)
		_, ok := det.Detect(sig)
		testkit.False(t, ok, "≤1 non-error result")
	})

	t.Run("rejects variadic signatures", func(t *testing.T) {
		t.Parallel()
		sig := buildSig(t, `package p
import "context"
type I interface { F(ctx context.Context, a string, b int, cs ...string) error }
`)
		_, ok := det.Detect(sig)
		testkit.False(t, ok, "variadic guard")
	})

	t.Run("matches generic signature with 3+ type-param args", func(t *testing.T) {
		t.Parallel()
		sig := buildSig(t, `package p
import "context"
type I[A, B, C any] interface { F(ctx context.Context, a A, b B, c C) error }
`)
		info, ok := det.Detect(sig)
		testkit.True(t, ok, "Detect")
		testkit.Equal(t, info.Shape, shape.MultiArgWriter, "Shape")
	})
}
