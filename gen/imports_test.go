// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package gen_test

import (
	"go/types"
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/gen"
)

func TestImportTracker(t *testing.T) {
	t.Parallel()

	t.Run("Add returns package name as qualifier", func(t *testing.T) {
		t.Parallel()
		tracker := gen.NewImportTracker("example.com/myapp/store")
		pkg := types.NewPackage("context", "context")
		alias := tracker.Add(pkg)
		testkit.Equal(t, alias, "context", "must return package name")
	})

	t.Run("Add returns empty for local package", func(t *testing.T) {
		t.Parallel()
		tracker := gen.NewImportTracker("example.com/myapp/store")
		pkg := types.NewPackage("example.com/myapp/store", "store")
		alias := tracker.Add(pkg)
		testkit.Equal(t, alias, "", "must return empty for local package")
	})

	t.Run("Add returns empty for nil package", func(t *testing.T) {
		t.Parallel()
		tracker := gen.NewImportTracker("example.com/myapp/store")
		alias := tracker.Add(nil)
		testkit.Equal(t, alias, "", "must return empty for nil")
	})

	t.Run("Add deduplicates same package", func(t *testing.T) {
		t.Parallel()
		tracker := gen.NewImportTracker("example.com/myapp/store")
		pkg := types.NewPackage("context", "context")
		a1 := tracker.Add(pkg)
		a2 := tracker.Add(pkg)
		testkit.Equal(t, a1, a2, "same package must return same alias")
		testkit.Len(t, tracker.Imports(), 1, "must have one import")
	})

	t.Run("Add assigns alias on name collision", func(t *testing.T) {
		t.Parallel()
		tracker := gen.NewImportTracker("example.com/myapp/store")
		pkg1 := types.NewPackage("example.com/v1/model", "model")
		pkg2 := types.NewPackage("example.com/v2/model", "model")
		a1 := tracker.Add(pkg1)
		a2 := tracker.Add(pkg2)
		testkit.Equal(t, a1, "model", "first gets base name")
		testkit.Equal(t, a2, "model2", "second gets numbered alias")
	})

	t.Run("AddPath registers by path", func(t *testing.T) {
		t.Parallel()
		tracker := gen.NewImportTracker("example.com/myapp/store")
		alias := tracker.AddPath("fmt")
		testkit.Equal(t, alias, "fmt", "must derive name from path")
	})

	t.Run("AddPath returns empty for local package", func(t *testing.T) {
		t.Parallel()
		tracker := gen.NewImportTracker("example.com/myapp/store")
		alias := tracker.AddPath("example.com/myapp/store")
		testkit.Equal(t, alias, "", "must return empty for local")
	})

	t.Run("AddPath deduplicates", func(t *testing.T) {
		t.Parallel()
		tracker := gen.NewImportTracker("example.com/myapp/store")
		a1 := tracker.AddPath("testing")
		a2 := tracker.AddPath("testing")
		testkit.Equal(t, a1, a2, "must deduplicate")
	})

	t.Run("Imports returns sorted by path", func(t *testing.T) {
		t.Parallel()
		tracker := gen.NewImportTracker("example.com/myapp/store")
		tracker.AddPath("testing")
		tracker.AddPath("context")
		tracker.AddPath("fmt")
		imports := tracker.Imports()
		paths := make([]string, len(imports))
		for i, imp := range imports {
			paths[i] = imp.Path
		}
		testkit.Equal(t, paths, []string{"context", "fmt", "testing"}, "must be sorted")
	})

	t.Run("Imports excludes local package", func(t *testing.T) {
		t.Parallel()
		tracker := gen.NewImportTracker("example.com/myapp/store")
		tracker.AddPath("example.com/myapp/store")
		tracker.AddPath("context")
		testkit.Len(t, tracker.Imports(), 1, "must exclude local")
	})

	t.Run("Import alias set only when name differs from base path", func(t *testing.T) {
		t.Parallel()
		tracker := gen.NewImportTracker("example.com/myapp/store")
		pkg1 := types.NewPackage("example.com/v1/model", "model")
		pkg2 := types.NewPackage("example.com/v2/model", "model")
		tracker.Add(pkg1)
		tracker.Add(pkg2)
		imports := tracker.Imports()
		testkit.Len(t, imports, 2, "must have 2 imports")
		// First model has no alias (name matches base).
		// Second model has alias "model2" (differs from base "model").
		for _, imp := range imports {
			if imp.Path == "example.com/v1/model" {
				testkit.Equal(t, imp.Alias, "", "first model needs no alias")
			}
			if imp.Path == "example.com/v2/model" {
				testkit.Equal(t, imp.Alias, "model2", "second model needs alias")
			}
		}
	})

	t.Run("Qualifier integrates with types.TypeString", func(t *testing.T) {
		t.Parallel()
		tracker := gen.NewImportTracker("example.com/myapp/store")
		pkg := types.NewPackage("context", "context")
		qual := tracker.Qualifier()
		result := qual(pkg)
		testkit.Equal(t, result, "context", "qualifier must call Add")
		testkit.Len(t, tracker.Imports(), 1, "must register import")
	})
}
