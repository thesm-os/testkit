// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package shape_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/generator/shape"
)

func TestPredicateDetector(t *testing.T) {
	t.Parallel()
	det := detectorByName(t, "Predicate")

	t.Run("matches func() bool", func(t *testing.T) {
		t.Parallel()
		sig := buildSig(t, `package p
type I interface { F() bool }
`)
		info, ok := det.Detect(sig)
		testkit.True(t, ok, "Detect")
		testkit.Equal(t, info.Shape, shape.Predicate, "Shape")
	})

	t.Run("rejects non-bool single result", func(t *testing.T) {
		t.Parallel()
		sig := buildSig(t, `package p
type I interface { F() int }
`)
		_, ok := det.Detect(sig)
		testkit.False(t, ok, "non-bool result rejected")
	})

	t.Run("rejects when ctx is present", func(t *testing.T) {
		t.Parallel()
		sig := buildSig(t, `package p
import "context"
type I interface { F(ctx context.Context) bool }
`)
		_, ok := det.Detect(sig)
		testkit.False(t, ok, "ctx disqualifies")
	})

	t.Run("rejects when params are present", func(t *testing.T) {
		t.Parallel()
		sig := buildSig(t, `package p
type I interface { F(k string) bool }
`)
		_, ok := det.Detect(sig)
		testkit.False(t, ok, "non-ctx param disqualifies")
	})

	t.Run("rejects when error is present", func(t *testing.T) {
		t.Parallel()
		sig := buildSig(t, `package p
type I interface { F() (bool, error) }
`)
		_, ok := det.Detect(sig)
		testkit.False(t, ok, "error result disqualifies")
	})

	t.Run("rejects variadic signatures", func(t *testing.T) {
		t.Parallel()
		sig := buildSig(t, `package p
type I interface { F(ks ...string) bool }
`)
		_, ok := det.Detect(sig)
		testkit.False(t, ok, "variadic guard")
	})
}
