// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package shape_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/generator/shape"
)

func TestStreamReaderDetector(t *testing.T) {
	t.Parallel()
	det := detectorByName(t, "StreamReader")

	t.Run("matches iter.Seq[V] return", func(t *testing.T) {
		t.Parallel()
		sig := buildSig(t, `package p
import (
	"context"
	"iter"
)
type I interface { F(ctx context.Context) iter.Seq[int] }
`)
		info, ok := det.Detect(sig)
		testkit.True(t, ok, "Detect")
		testkit.Equal(t, info.Shape, shape.StreamReader, "Shape")
	})

	t.Run("matches iter.Seq2[V, error] return", func(t *testing.T) {
		t.Parallel()
		sig := buildSig(t, `package p
import (
	"context"
	"iter"
)
type I interface { F(ctx context.Context) iter.Seq2[int, error] }
`)
		_, ok := det.Detect(sig)
		testkit.True(t, ok, "iter.Seq2")
	})

	t.Run("matches without ctx", func(t *testing.T) {
		t.Parallel()
		sig := buildSig(t, `package p
import "iter"
type I interface { F() iter.Seq[int] }
`)
		_, ok := det.Detect(sig)
		testkit.True(t, ok, "no-ctx StreamReader")
	})

	t.Run("rejects non-iter return types", func(t *testing.T) {
		t.Parallel()
		sig := buildSig(t, `package p
type I interface { F() int }
`)
		_, ok := det.Detect(sig)
		testkit.False(t, ok, "non-iter rejected")
	})

	t.Run("rejects when no return", func(t *testing.T) {
		t.Parallel()
		sig := buildSig(t, `package p
type I interface { F() }
`)
		_, ok := det.Detect(sig)
		testkit.False(t, ok, "void return rejected")
	})

	t.Run("matches generic signature with iter.Seq2[V, error]", func(t *testing.T) {
		t.Parallel()
		sig := buildSig(t, `package p
import (
	"context"
	"iter"
)
type I[V any] interface { F(ctx context.Context) iter.Seq2[V, error] }
`)
		info, ok := det.Detect(sig)
		testkit.True(t, ok, "Detect")
		testkit.Equal(t, info.Shape, shape.StreamReader, "Shape")
	})
}
