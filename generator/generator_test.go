// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package generator_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/generator"
)

type fakeGenerator struct{ name string }

func (f *fakeGenerator) Name() string { return f.name }
func (*fakeGenerator) Generate(
	_ *generator.Package, _ []string, _ generator.Config, _ generator.Options,
) (*generator.Result, error) {
	return &generator.Result{}, nil
}

func TestRegistry(t *testing.T) {
	t.Parallel()

	t.Run("Register stores by name and Get retrieves it", func(t *testing.T) {
		t.Parallel()
		r := generator.NewRegistry()
		g := &fakeGenerator{name: "stub"}

		testkit.NoError(t, r.Register(g), "first register")
		testkit.True(t, r.Get("stub") == g, "Get returns the registered generator")
		testkit.True(t, r.Get("missing") == nil, "Get on missing returns nil")
		testkit.Equal(t, r.Len(), 1, "Len reflects registered count")
	})

	t.Run("Register rejects duplicates", func(t *testing.T) {
		t.Parallel()
		r := generator.NewRegistry()
		_ = r.Register(&fakeGenerator{name: "stub"})
		err := r.Register(&fakeGenerator{name: "stub"})
		testkit.True(t, err != nil, "duplicate registration must error")
	})

	t.Run("Names returns sorted output", func(t *testing.T) {
		t.Parallel()
		r := generator.NewRegistry()
		_ = r.Register(&fakeGenerator{name: "suite"})
		_ = r.Register(&fakeGenerator{name: "stub"})
		_ = r.Register(&fakeGenerator{name: "bench"})

		names := r.Names()
		testkit.Len(t, names, 3, "three registered")
		testkit.Equal(t, names[0], "bench", "alphabetical order")
		testkit.Equal(t, names[1], "stub", "alphabetical order")
		testkit.Equal(t, names[2], "suite", "alphabetical order")
	})

	t.Run("MustRegister panics on duplicate", func(t *testing.T) {
		t.Parallel()
		r := generator.NewRegistry()
		r.MustRegister(&fakeGenerator{name: "stub"})
		defer func() {
			testkit.True(t, recover() != nil, "duplicate MustRegister must panic")
		}()
		r.MustRegister(&fakeGenerator{name: "stub"})
	})
}
