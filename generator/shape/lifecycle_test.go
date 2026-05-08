// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package shape_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/generator/shape"
)

func TestLifecycleDetector(t *testing.T) {
	t.Parallel()
	det := detectorByName(t, "Lifecycle")

	t.Run("matches func(ctx) error", func(t *testing.T) {
		t.Parallel()
		sig := buildSig(t, `package p
import "context"
type I interface { F(ctx context.Context) error }
`)
		info, ok := det.Detect(sig)
		testkit.True(t, ok, "Detect")
		testkit.Equal(t, info.Shape, shape.Lifecycle, "Shape")
	})

	t.Run("rejects when ctx is missing", func(t *testing.T) {
		t.Parallel()
		sig := buildSig(t, `package p
type I interface { F() error }
`)
		_, ok := det.Detect(sig)
		testkit.False(t, ok, "ctx-required guard")
	})

	t.Run("rejects when non-ctx params are present", func(t *testing.T) {
		t.Parallel()
		sig := buildSig(t, `package p
import "context"
type I interface { F(ctx context.Context, k string) error }
`)
		_, ok := det.Detect(sig)
		testkit.False(t, ok, "Writer-shape rejected")
	})

	t.Run("rejects when no error result", func(t *testing.T) {
		t.Parallel()
		sig := buildSig(t, `package p
import "context"
type I interface { F(ctx context.Context) }
`)
		_, ok := det.Detect(sig)
		testkit.False(t, ok, "VoidLifecycle-shape rejected")
	})

	t.Run("rejects when non-error results present", func(t *testing.T) {
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
type I interface { F(ctx context.Context, ks ...string) error }
`)
		_, ok := det.Detect(sig)
		testkit.False(t, ok, "variadic guard")
	})
}
