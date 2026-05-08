// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package shape_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/generator/directive"
	"go.thesmos.sh/testkit/generator/shape"
)

func TestDeleterDetector(t *testing.T) {
	t.Parallel()
	det := detectorByName(t, "Deleter")

	t.Run("matches when //testkit:deleter directive is present", func(t *testing.T) {
		t.Parallel()
		sig := buildSig(t, `package p
import "context"
type I interface { F(ctx context.Context, k string) error }
`, directive.Directive{Name: directive.Deleter})
		info, ok := det.Detect(sig)
		testkit.True(t, ok, "Detect")
		testkit.Equal(t, info.Shape, shape.Deleter, "Shape")
	})

	t.Run("rejects without the directive", func(t *testing.T) {
		t.Parallel()
		sig := buildSig(t, `package p
import "context"
type I interface { F(ctx context.Context, k string) error }
`)
		_, ok := det.Detect(sig)
		testkit.False(t, ok, "directive is required")
	})

	t.Run("rejects wrong-shape signatures even with directive", func(t *testing.T) {
		t.Parallel()
		sig := buildSig(t, `package p
import "context"
type I interface { F(ctx context.Context, k string) (int, error) }
`, directive.Directive{Name: directive.Deleter})
		_, ok := det.Detect(sig)
		testkit.False(t, ok, "Reader-shape rejected even with directive")
	})

	t.Run("rejects variadic signatures", func(t *testing.T) {
		t.Parallel()
		sig := buildSig(t, `package p
import "context"
type I interface { F(ctx context.Context, ks ...string) error }
`, directive.Directive{Name: directive.Deleter})
		_, ok := det.Detect(sig)
		testkit.False(t, ok, "variadic guard")
	})

	t.Run("rejects 0 non-ctx params", func(t *testing.T) {
		t.Parallel()
		sig := buildSig(t, `package p
import "context"
type I interface { F(ctx context.Context) error }
`, directive.Directive{Name: directive.Deleter})
		_, ok := det.Detect(sig)
		testkit.False(t, ok, "needs key param")
	})

	t.Run("matches generic signature with directive", func(t *testing.T) {
		t.Parallel()
		sig := buildSig(t, `package p
import "context"
type I[K comparable] interface { F(ctx context.Context, k K) error }
`, directive.Directive{Name: directive.Deleter})
		info, ok := det.Detect(sig)
		testkit.True(t, ok, "Detect")
		testkit.Equal(t, info.Shape, shape.Deleter, "Shape")
	})
}
