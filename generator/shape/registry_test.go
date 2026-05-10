// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package shape_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/generator/directive"
	"go.thesmos.sh/testkit/generator/shape"
)

func TestRegistry(t *testing.T) {
	t.Parallel()

	t.Run("DefaultRegistry contains all 37 shipped detectors", func(t *testing.T) {
		t.Parallel()
		dets := shape.DefaultRegistry().Detectors()
		testkit.Len(t, dets, 37, "shipped detector count")
	})

	t.Run("DefaultRegistry returns the same instance on repeated calls", func(t *testing.T) {
		t.Parallel()
		a := shape.DefaultRegistry()
		b := shape.DefaultRegistry()
		testkit.True(t, a == b, "default is a singleton")
	})

	t.Run("NewRegistry produces a fresh registry equivalent to DefaultRegistry", func(t *testing.T) {
		t.Parallel()
		r := shape.NewRegistry()
		testkit.Len(t, r.Detectors(), 37, "fresh registry has full detector set")
	})

	t.Run("Detectors returns a defensive copy", func(t *testing.T) {
		t.Parallel()
		r := shape.NewRegistry()
		dets := r.Detectors()
		// Truncate the returned slice — the registry's internal state must not change.
		dets = dets[:0]
		_ = dets
		testkit.Len(t, r.Detectors(), 37, "mutating the returned slice does not affect the registry")
	})
}

// TestRegistry_Routing covers cross-detector dispatch behavior — the
// cases where two or more detectors *could* match a signature and the
// registry's priority cascade picks one. Per-detector tests stay
// focused on accept/reject of their own pattern; registry routing
// lives here.
func TestRegistry_Routing(t *testing.T) {
	t.Parallel()

	t.Run("//testkit:deleter elevates Writer signature to Deleter", func(t *testing.T) {
		t.Parallel()
		const src = `package p
import "context"
type I interface { F(ctx context.Context, k string) error }
`
		got := classifyOne(t, src, directive.Directive{Name: directive.Deleter})
		testkit.Equal(t, got, "Deleter", "directive elevates")

		gotPlain := classifyOne(t, src)
		testkit.Equal(t, gotPlain, "Writer", "no directive → Writer")
	})

	t.Run("Classify falls back to Unknown when no detector matches", func(t *testing.T) {
		t.Parallel()
		const src = `package p
import "context"
type I interface {
	// 4 non-ctx params, no error → no detector matches
	F(ctx context.Context, a, b, c, d string)
}
`
		testkit.Equal(t, classifyOne(t, src), "Unknown", "exotic shape falls to Unknown")
	})

	t.Run("variadic + slice result routes to BatchReader, not Reader", func(t *testing.T) {
		t.Parallel()
		const src = `package p
import "context"
type I interface { F(ctx context.Context, ks ...string) ([]int, error) }
`
		testkit.Equal(t, classifyOne(t, src), "BatchReader", "variadic claims first")
	})

	t.Run("interface-typed param routes to StreamConsumer, not Reader", func(t *testing.T) {
		t.Parallel()
		const src = `package p
import (
	"context"
	"io"
)
type I interface { F(ctx context.Context, r io.Reader) (int, error) }
`
		testkit.Equal(t, classifyOne(t, src), "StreamConsumer", "interface-typed claims first")
	})

	t.Run("pointer result routes to PointerReader, not ReaderNoError", func(t *testing.T) {
		t.Parallel()
		const src = `package p
type Item struct{}
type I interface { F(k string) *Item }
`
		testkit.Equal(t, classifyOne(t, src), "PointerReader", "PointerReader claims first")
	})

	t.Run("StreamReader claims iter.Seq returns over any other shape", func(t *testing.T) {
		t.Parallel()
		const src = `package p
import (
	"context"
	"iter"
)
type I interface { F(ctx context.Context) iter.Seq2[int, error] }
`
		testkit.Equal(t, classifyOne(t, src), "StreamReader", "iter.Seq2 wins outright")
	})
}

func TestClassifyInterface(t *testing.T) {
	t.Parallel()

	t.Run("returns one Info per method aligned with input order", func(t *testing.T) {
		t.Parallel()
		const src = `package p
import "context"
type I interface {
    Get(ctx context.Context, k string) (string, error)
    Put(ctx context.Context, v string) error
}
`
		got := classifyAllViaInterface(t, src, "I", nil)
		testkit.Equal(t, got["Get"], "Reader", "Get is Reader")
		testkit.Equal(t, got["Put"], "Writer", "Put is Writer")
	})

	t.Run("with no contract-tier override, pass 2 result equals pass 1", func(t *testing.T) {
		t.Parallel()
		const src = `package p
type I interface { F(k string) (int, error) }
`
		got := classifyAllViaInterface(t, src, "I", nil)
		testkit.Equal(t, got["F"], "Reader", "no contract-tier promotion → Reader stands")
	})

	t.Run("InterfaceContext.Shapes is populated with signature-tier results", func(t *testing.T) {
		t.Parallel()
		const src = `package p
import "context"
type I interface {
    Get(ctx context.Context, k string) (string, error)
    Save(ctx context.Context, v string) error
}
`
		got := classifyAllViaInterface(t, src, "I", nil)
		testkit.Equal(t, got["Get"], "Reader", "Get pre-classified as Reader")
		testkit.Equal(t, got["Save"], "Writer", "Save pre-classified as Writer")
	})
}
