// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package gen_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/gen"
)

func TestDirectives(t *testing.T) {
	t.Parallel()
	pkg := loadTestPackage(t, "directives")

	t.Run("type with directive", func(t *testing.T) {
		t.Parallel()
		dirs := pkg.Directives("Item")
		testkit.Len(t, dirs, 1, "must have 1 directive")
		testkit.Equal(t, dirs[0].Name, "immutable", "directive name")
	})

	t.Run("type with no directives returns nil", func(t *testing.T) {
		t.Parallel()
		dirs := pkg.Directives("Status")
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

	t.Run("concrete method directive", func(t *testing.T) {
		t.Parallel()
		concretePkg := loadTestPackage(t, "concrete")
		dirs := concretePkg.MethodDirectives("Service", "Run")
		testkit.Len(t, dirs, 1, "must have 1 directive")
		testkit.Equal(t, dirs[0].Name, "timeout", "directive name")
		testkit.Equal(t, dirs[0].Args, []string{"5s"}, "directive args")
	})

	t.Run("concrete method without directives", func(t *testing.T) {
		t.Parallel()
		concretePkg := loadTestPackage(t, "concrete")
		dirs := concretePkg.MethodDirectives("Service", "Stop")
		testkit.Len(t, dirs, 0, "must have no directives")
	})
}

func TestFieldDirectives(t *testing.T) {
	t.Parallel()
	pkg := loadTestPackage(t, "directives")

	t.Run("field with directive", func(t *testing.T) {
		t.Parallel()
		dirs := pkg.FieldDirectives("Item", "ID")
		testkit.Len(t, dirs, 1, "must have 1 directive")
		testkit.Equal(t, dirs[0].Name, "optional", "directive name")
	})

	t.Run("field with no directives", func(t *testing.T) {
		t.Parallel()
		dirs := pkg.FieldDirectives("Item", "Name")
		testkit.Len(t, dirs, 0, "must have no directives")
	})

	t.Run("nonexistent field returns nil", func(t *testing.T) {
		t.Parallel()
		dirs := pkg.FieldDirectives("Item", "Nonexistent")
		testkit.True(t, dirs == nil, "must return nil")
	})

	t.Run("nonexistent type returns nil", func(t *testing.T) {
		t.Parallel()
		dirs := pkg.FieldDirectives("Nonexistent", "ID")
		testkit.True(t, dirs == nil, "must return nil")
	})

	t.Run("interface type returns nil", func(t *testing.T) {
		t.Parallel()
		dirs := pkg.FieldDirectives("Store", "Get")
		testkit.True(t, dirs == nil, "must return nil for non-struct")
	})
}

func TestVarDirectives(t *testing.T) {
	t.Parallel()
	pkg := loadTestPackage(t, "directives")

	t.Run("var with directive", func(t *testing.T) {
		t.Parallel()
		dirs := pkg.VarDirectives("ErrNotFound")
		testkit.Len(t, dirs, 1, "must have 1 directive")
		testkit.Equal(t, dirs[0].Name, "sentinel", "directive name")
	})

	t.Run("var with no directives", func(t *testing.T) {
		t.Parallel()
		dirs := pkg.VarDirectives("ErrConflict")
		testkit.Len(t, dirs, 0, "must have no directives")
	})

	t.Run("nonexistent var returns nil", func(t *testing.T) {
		t.Parallel()
		dirs := pkg.VarDirectives("Nonexistent")
		testkit.True(t, dirs == nil, "must return nil")
	})
}

func TestConstDirectives(t *testing.T) {
	t.Parallel()
	pkg := loadTestPackage(t, "directives")

	t.Run("const with directive", func(t *testing.T) {
		t.Parallel()
		dirs := pkg.ConstDirectives("StatusPending")
		testkit.Len(t, dirs, 1, "must have 1 directive")
		testkit.Equal(t, dirs[0].Name, "default", "directive name")
	})

	t.Run("const with no directives", func(t *testing.T) {
		t.Parallel()
		dirs := pkg.ConstDirectives("StatusActive")
		testkit.Len(t, dirs, 0, "must have no directives")
	})

	t.Run("nonexistent const returns nil", func(t *testing.T) {
		t.Parallel()
		dirs := pkg.ConstDirectives("Nonexistent")
		testkit.True(t, dirs == nil, "must return nil")
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
		var stub gen.GenerateDirective
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
		var builder gen.GenerateDirective
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
