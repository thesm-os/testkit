// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package generator_test

import (
	"go/types"
	"strconv"
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/generator"
)

func TestImportTracker(t *testing.T) {
	t.Parallel()

	t.Run("local package returns empty alias", func(t *testing.T) {
		t.Parallel()
		tr := generator.NewImportTracker("mypkg/store")
		testkit.Equal(t, tr.AddPath("mypkg/store"), "", "local pkg → bare reference")
		testkit.Equal(t, tr.AddPath(""), "", "empty path → bare reference")
		testkit.Len(t, tr.Imports(), 0, "no imports recorded")
	})

	t.Run("first registration uses basename", func(t *testing.T) {
		t.Parallel()
		tr := generator.NewImportTracker("mypkg/store")
		testkit.Equal(t, tr.AddPath("context"), "context", "context's basename")
		testkit.Equal(t, tr.AddPath("io"), "io", "io's basename")
		imps := tr.Imports()
		testkit.Len(t, imps, 2, "two imports tracked")
		testkit.Equal(t, imps[0].Path, "context", "imports sorted by path")
		testkit.Equal(t, imps[1].Path, "io", "imports sorted by path")
	})

	t.Run("collisions get numbered aliases", func(t *testing.T) {
		t.Parallel()
		tr := generator.NewImportTracker("mypkg/store")
		testkit.Equal(t, tr.AddPath("a/model"), "model", "first model")
		testkit.Equal(t, tr.AddPath("b/model"), "model2", "second model")
		testkit.Equal(t, tr.AddPath("c/model"), "model3", "third model")
		testkit.Equal(t, tr.AddPath("b/model"), "model2", "re-add returns same alias")
	})

	t.Run("LocalPkg accessor returns the local path", func(t *testing.T) {
		t.Parallel()
		tr := generator.NewImportTracker("mypkg/store")
		testkit.Equal(t, tr.LocalPkg(), "mypkg/store", "local pkg path")
	})

	t.Run("Add via *types.Package routes through AddPath", func(t *testing.T) {
		t.Parallel()
		tr := generator.NewImportTracker("mypkg/store")

		ctxPkg := types.NewPackage("context", "context")
		testkit.Equal(t, tr.Add(ctxPkg), "context", "stdlib package")

		// nil package is a no-op.
		testkit.Equal(t, tr.Add(nil), "", "nil package → empty alias")
	})

	t.Run("Qualifier returns alias appropriate for templates", func(t *testing.T) {
		t.Parallel()
		tr := generator.NewImportTracker("mypkg/store")
		ctxPkg := types.NewPackage("context", "context")
		q := tr.Qualifier()
		testkit.Equal(t, q(ctxPkg), "context", "Qualifier emits package alias")

		localPkg := types.NewPackage("mypkg/store", "store")
		testkit.Equal(t, q(localPkg), "", "local pkg → bare reference")
		testkit.Equal(t, q(nil), "", "nil → bare reference")
	})
}

func TestImport(t *testing.T) {
	t.Parallel()

	t.Run("String renders bare path when alias matches basename", func(t *testing.T) {
		t.Parallel()
		i := generator.Import{Path: "context"}
		testkit.Equal(t, i.String(), strconv.Quote("context"), "no alias prefix")
	})

	t.Run("String renders alias prefix when set", func(t *testing.T) {
		t.Parallel()
		i := generator.Import{Alias: "model2", Path: "b/model"}
		testkit.Equal(t, i.String(), `model2 `+strconv.Quote("b/model"), "alias-prefixed import")
	})
}
