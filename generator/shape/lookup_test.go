// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package shape_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/generator/shape"
)

func TestLookupDetector(t *testing.T) {
	t.Parallel()
	det := detectorByName(t, "Lookup")

	t.Run("matches func(K) (R1, R2, bool)", func(t *testing.T) {
		t.Parallel()
		sig := buildSig(t, `package p
type I interface { F(k string) (int, string, bool) }
`)
		info, ok := det.Detect(sig)
		testkit.True(t, ok, "Detect")
		testkit.Equal(t, info.Shape, shape.Lookup, "Shape")
	})

	t.Run("matches with ctx", func(t *testing.T) {
		t.Parallel()
		sig := buildSig(t, `package p
import "context"
type I interface { F(ctx context.Context, k string) (int, string, bool) }
`)
		_, ok := det.Detect(sig)
		testkit.True(t, ok, "ctx + 3 results, last bool")
	})

	t.Run("rejects when last result is not bool", func(t *testing.T) {
		t.Parallel()
		sig := buildSig(t, `package p
type I interface { F(k string) (int, string, error) }
`)
		_, ok := det.Detect(sig)
		testkit.False(t, ok, "trailing error rejected")
	})

	t.Run("rejects 2-result tuples", func(t *testing.T) {
		t.Parallel()
		sig := buildSig(t, `package p
type I interface { F(k string) (int, bool) }
`)
		_, ok := det.Detect(sig)
		testkit.False(t, ok, "2 results disqualifies")
	})

	t.Run("rejects variadic signatures", func(t *testing.T) {
		t.Parallel()
		sig := buildSig(t, `package p
type I interface { F(ks ...string) (int, string, bool) }
`)
		_, ok := det.Detect(sig)
		testkit.False(t, ok, "variadic guard")
	})

	t.Run("rejects when 0 non-ctx params", func(t *testing.T) {
		t.Parallel()
		sig := buildSig(t, `package p
import "context"
type I interface { F(ctx context.Context) (int, string, bool) }
`)
		_, ok := det.Detect(sig)
		testkit.False(t, ok, "no key disqualifies")
	})

	t.Run("matches generic signature", func(t *testing.T) {
		t.Parallel()
		sig := buildSig(t, `package p
type I[K comparable, V any, M any] interface { F(k K) (V, M, bool) }
`)
		info, ok := det.Detect(sig)
		testkit.True(t, ok, "Detect")
		testkit.Equal(t, info.Shape, shape.Lookup, "Shape")
	})
}
