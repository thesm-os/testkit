// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package gen

import (
	"testing"

	"go.thesmos.sh/testkit"
)

func TestDirectives(t *testing.T) {
	t.Parallel()
	pkg := loadTestPackage(t, "directives")

	t.Run("type with no directives returns nil", func(t *testing.T) {
		t.Parallel()
		dirs := pkg.Directives("Item")
		testkit.True(t, dirs == nil, "must return nil for undirected type")
	})

	t.Run("nonexistent type returns nil", func(t *testing.T) {
		t.Parallel()
		dirs := pkg.Directives("Nonexistent")
		testkit.True(t, dirs == nil, "must return nil for missing type")
	})
}

func TestMethodDirectives(t *testing.T) {
	t.Parallel()
	pkg := loadTestPackage(t, "directives")

	t.Run("Get has errors and idempotent directives", func(t *testing.T) {
		t.Parallel()
		dirs := pkg.MethodDirectives("Store", "Get")
		testkit.Len(t, dirs, 2, "must have 2 directives")
		testkit.Equal(t, dirs[0].Name, "errors", "first directive name")
		testkit.Equal(t, dirs[0].Args, []string{"ErrNotFound", "ErrConflict"}, "errors args")
		testkit.Equal(t, dirs[1].Name, "idempotent", "second directive name")
		testkit.Len(t, dirs[1].Args, 0, "idempotent has no args")
	})

	t.Run("Put has errors and concurrent directives", func(t *testing.T) {
		t.Parallel()
		dirs := pkg.MethodDirectives("Store", "Put")
		testkit.Len(t, dirs, 2, "must have 2 directives")
		testkit.Equal(t, dirs[0].Name, "errors", "first directive name")
		testkit.Equal(t, dirs[0].Args, []string{"ErrConflict"}, "errors args")
		testkit.Equal(t, dirs[1].Name, "concurrent", "second directive name")
	})

	t.Run("Delete has no directives", func(t *testing.T) {
		t.Parallel()
		dirs := pkg.MethodDirectives("Store", "Delete")
		testkit.Len(t, dirs, 0, "must have no directives")
	})

	t.Run("nonexistent method returns nil", func(t *testing.T) {
		t.Parallel()
		dirs := pkg.MethodDirectives("Store", "Nonexistent")
		testkit.True(t, dirs == nil, "must return nil for missing method")
	})

	t.Run("nonexistent type returns nil", func(t *testing.T) {
		t.Parallel()
		dirs := pkg.MethodDirectives("Nonexistent", "Get")
		testkit.True(t, dirs == nil, "must return nil for missing type")
	})
}

func TestGenerateDirectives(t *testing.T) {
	t.Parallel()
	pkg := loadTestPackage(t, "directives")

	t.Run("returns all go:generate testkit directives", func(t *testing.T) {
		t.Parallel()
		dirs := pkg.GenerateDirectives()
		testkit.Len(t, dirs, 3, "must find 3 directives")
	})

	t.Run("parses stub directive", func(t *testing.T) {
		t.Parallel()
		dirs := pkg.GenerateDirectives()
		// Find the stub directive.
		var stub GenerateDirective
		for _, d := range dirs {
			if d.Generator == "stub" {
				stub = d
				break
			}
		}
		testkit.Equal(t, stub.Generator, "stub", "generator must be stub")
		testkit.Equal(t, stub.Output, "storetest/in_memory_store.gen.go", "output path")
		testkit.Equal(t, stub.Types, []string{"Store"}, "types")
	})

	t.Run("parses builder directive with output", func(t *testing.T) {
		t.Parallel()
		dirs := pkg.GenerateDirectives()
		var builder GenerateDirective
		for _, d := range dirs {
			if d.Generator == "builder" {
				builder = d
				break
			}
		}
		testkit.Equal(t, builder.Generator, "builder", "generator")
		testkit.Equal(t, builder.Output, "storetest/builders.gen.go", "output")
		testkit.Equal(t, builder.Types, []string{"Item"}, "types")
	})

	t.Run("directives have file and line info", func(t *testing.T) {
		t.Parallel()
		dirs := pkg.GenerateDirectives()
		for _, d := range dirs {
			testkit.Assert(t, d.File).IsNotEmpty("must have filename")
			testkit.True(t, d.Line > 0, "must have line number")
		}
	})
}

func TestParseGenerateDirective(t *testing.T) {
	t.Parallel()

	t.Run("empty body returns empty directive", func(t *testing.T) {
		t.Parallel()
		d := parseGenerateDirective("//go:generate testkit ")
		testkit.Equal(t, d.Generator, "", "must be empty")
	})

	t.Run("subcommand only", func(t *testing.T) {
		t.Parallel()
		d := parseGenerateDirective("//go:generate testkit sentinel")
		testkit.Equal(t, d.Generator, "sentinel", "generator")
		testkit.Len(t, d.Types, 0, "no types")
		testkit.Equal(t, d.Output, "", "no output")
	})

	t.Run("skips unknown flags", func(t *testing.T) {
		t.Parallel()
		d := parseGenerateDirective("//go:generate testkit stub -v -o out.go Store")
		testkit.Equal(t, d.Generator, "stub", "generator")
		testkit.Equal(t, d.Output, "out.go", "output")
		testkit.Equal(t, d.Types, []string{"Store"}, "types")
	})
}
