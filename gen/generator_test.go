// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package gen_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/gen"
)

type fakeGenerator struct {
	name   string
	called bool
}

func (g *fakeGenerator) Name() string { return g.name }

func (g *fakeGenerator) Generate(
	_ *gen.Package, _ []string, _ gen.Config, _ gen.Options,
) (*gen.Result, error) {
	g.called = true
	return &gen.Result{}, nil
}

func TestRegistry(t *testing.T) {
	t.Parallel()

	t.Run("Register and Get", func(t *testing.T) {
		t.Parallel()
		r := gen.NewRegistry()
		g := &fakeGenerator{name: "stub"}
		r.Register(g)

		got := r.Get("stub")
		testkit.True(t, got == g, "must return registered generator")
	})

	t.Run("Get returns nil for unknown", func(t *testing.T) {
		t.Parallel()
		r := gen.NewRegistry()
		testkit.True(t, r.Get("nonexistent") == nil, "must return nil")
	})

	t.Run("Names returns sorted list", func(t *testing.T) {
		t.Parallel()
		r := gen.NewRegistry()
		r.Register(&fakeGenerator{name: "stub"})
		r.Register(&fakeGenerator{name: "builder"})
		r.Register(&fakeGenerator{name: "enum"})

		testkit.Equal(t, r.Names(), []string{"builder", "enum", "stub"}, "must be sorted")
	})

	t.Run("duplicate registration panics", func(t *testing.T) {
		t.Parallel()
		r := gen.NewRegistry()
		r.Register(&fakeGenerator{name: "stub"})

		testkit.Panics(t, func() {
			r.Register(&fakeGenerator{name: "stub"})
		}, "must panic on duplicate")
	})

	t.Run("empty registry has no names", func(t *testing.T) {
		t.Parallel()
		r := gen.NewRegistry()
		testkit.Len(t, r.Names(), 0, "empty registry")
	})
}
