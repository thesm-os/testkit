// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package shape_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/generator/shape"
)

func TestStreamConsumerDetector(t *testing.T) {
	t.Parallel()
	det := detectorByName(t, "StreamConsumer")

	t.Run("matches func(ctx, S) (V, error) when S is interface-typed", func(t *testing.T) {
		t.Parallel()
		sig := buildSig(t, `package p
import (
	"context"
	"io"
)
type I interface { F(ctx context.Context, r io.Reader) (int, error) }
`)
		info, ok := det.Detect(sig)
		testkit.True(t, ok, "Detect")
		testkit.Equal(t, info.Shape, shape.StreamConsumer, "Shape")
	})

	t.Run("rejects non-interface non-ctx param", func(t *testing.T) {
		t.Parallel()
		sig := buildSig(t, `package p
import "context"
type I interface { F(ctx context.Context, k string) (int, error) }
`)
		_, ok := det.Detect(sig)
		testkit.False(t, ok, "string is not interface")
	})

	t.Run("rejects when ctx is missing", func(t *testing.T) {
		t.Parallel()
		sig := buildSig(t, `package p
import "io"
type I interface { F(r io.Reader) (int, error) }
`)
		_, ok := det.Detect(sig)
		testkit.False(t, ok, "ctx-required guard")
	})

	t.Run("rejects variadic signatures", func(t *testing.T) {
		t.Parallel()
		sig := buildSig(t, `package p
import (
	"context"
	"io"
)
type I interface { F(ctx context.Context, rs ...io.Reader) (int, error) }
`)
		_, ok := det.Detect(sig)
		testkit.False(t, ok, "variadic guard")
	})

	t.Run("rejects when no error result", func(t *testing.T) {
		t.Parallel()
		sig := buildSig(t, `package p
import (
	"context"
	"io"
)
type I interface { F(ctx context.Context, r io.Reader) int }
`)
		_, ok := det.Detect(sig)
		testkit.False(t, ok, "missing error guard")
	})

	t.Run("rejects multi-result", func(t *testing.T) {
		t.Parallel()
		sig := buildSig(t, `package p
import (
	"context"
	"io"
)
type I interface { F(ctx context.Context, r io.Reader) (int, int, error) }
`)
		_, ok := det.Detect(sig)
		testkit.False(t, ok, "multi non-error guard")
	})

	// Regression: type parameters report their constraint as their
	// underlying type. `K comparable` has *types.Interface as its
	// underlying. Without explicit *types.TypeParam exclusion in
	// isInterfaceTyped, every generic Reader's K param would be
	// misclassified as a StreamConsumer. This test guards against the
	// regression.
	t.Run("rejects type-parameter K (constraint underlying must not match)", func(t *testing.T) {
		t.Parallel()
		sig := buildSig(t, `package p
import "context"
type I[K comparable, V any] interface { F(ctx context.Context, k K) (V, error) }
`)
		_, ok := det.Detect(sig)
		testkit.False(t, ok, "type param's constraint-as-underlying must not match StreamConsumer")
	})
}
