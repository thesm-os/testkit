// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package shape_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/generator/shape"
)

func TestReaderWithBoolDetector(t *testing.T) {
	t.Parallel()
	det := detectorByName(t, "ReaderWithBool")

	t.Run("matches func(K) (V, bool)", func(t *testing.T) {
		t.Parallel()
		sig := buildSig(t, `package p
type I interface { F(k string) (int, bool) }
`)
		info, ok := det.Detect(sig)
		testkit.True(t, ok, "Detect")
		testkit.Equal(t, info.Shape, shape.ReaderWithBool, "Shape")
	})

	t.Run("matches with ctx", func(t *testing.T) {
		t.Parallel()
		sig := buildSig(t, `package p
import "context"
type I interface { F(ctx context.Context, k string) (int, bool) }
`)
		_, ok := det.Detect(sig)
		testkit.True(t, ok, "ctx + (V, bool)")
	})

	t.Run("rejects when last result is not bool", func(t *testing.T) {
		t.Parallel()
		sig := buildSig(t, `package p
type I interface { F(k string) (int, error) }
`)
		_, ok := det.Detect(sig)
		testkit.False(t, ok, "(V, error) rejected")
	})

	t.Run("rejects 3-result tuples", func(t *testing.T) {
		t.Parallel()
		sig := buildSig(t, `package p
type I interface { F(k string) (int, string, bool) }
`)
		_, ok := det.Detect(sig)
		testkit.False(t, ok, "3 results disqualifies")
	})

	t.Run("rejects when 0 non-ctx params", func(t *testing.T) {
		t.Parallel()
		sig := buildSig(t, `package p
type I interface { F() (int, bool) }
`)
		_, ok := det.Detect(sig)
		testkit.False(t, ok, "no key disqualifies")
	})

	t.Run("rejects variadic signatures", func(t *testing.T) {
		t.Parallel()
		sig := buildSig(t, `package p
type I interface { F(ks ...string) (int, bool) }
`)
		_, ok := det.Detect(sig)
		testkit.False(t, ok, "variadic guard")
	})

	t.Run("matches generic signature", func(t *testing.T) {
		t.Parallel()
		sig := buildSig(t, `package p
type I[K comparable, V any] interface { F(k K) (V, bool) }
`)
		info, ok := det.Detect(sig)
		testkit.True(t, ok, "Detect")
		testkit.Equal(t, info.Shape, shape.ReaderWithBool, "Shape")
	})
}
