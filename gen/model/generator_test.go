// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package model_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/gen"
	"go.thesmos.sh/testkit/gen/model"
)

func testdataDir(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot determine test file location")
	}
	return filepath.Join(filepath.Dir(file), "testdata")
}

func loadTestPackage(t *testing.T, subdir string) *gen.Package {
	t.Helper()
	loader := gen.NewLoader()
	dir := filepath.Join(testdataDir(t), subdir)
	pkg, err := loader.Load(".", dir)
	testkit.NoError(t, err, "must load package")
	return pkg
}

func TestGenerate(t *testing.T) {
	t.Parallel()

	t.Run("Name returns model", func(t *testing.T) {
		t.Parallel()
		g := &model.Generator{}
		testkit.Equal(t, g.Name(), "model", "generator name")
	})

	t.Run("renders basic Store", func(t *testing.T) {
		t.Parallel()
		pkg := loadTestPackage(t, "basic")
		g := &model.Generator{}
		result, err := g.Generate(pkg, []string{"Store"}, gen.DefaultConfig(), gen.Options{
			Output: "storetest/store_model.gen.go",
		})
		testkit.NoError(t, err, "must generate")
		content := string(result.Files[0].Content)
		testkit.Assert(t, content).Contains("AssertStoreModel", "must have entry point")
		testkit.Assert(t, content).Contains("action.Reader", "must use Reader action helper")
		testkit.Assert(t, content).Contains("action.Writer", "must use Writer action helper")
		testkit.Assert(t, content).Contains("ReadAfterWrite", "must have ReadAfterWrite law")
		testkit.Assert(t, content).Contains("refmap.NewMapStore", "must synthesize reference")
		testkit.Assert(t, content).Contains("StoreModelOption", "must have option type")
	})

	t.Run("golden/basic", func(t *testing.T) {
		t.Parallel()
		pkg := loadTestPackage(t, "basic")
		g := &model.Generator{}
		result, err := g.Generate(pkg, []string{"Store"}, gen.DefaultConfig(), gen.Options{
			Output: "storetest/store_model.gen.go",
		})
		testkit.NoError(t, err, "must generate")

		goldenFile := filepath.Join(testdataDir(t), "basic", "storetest", "store_model.gen.go")
		want, err := os.ReadFile(goldenFile)
		testkit.NoError(t, err, "must read golden file")
		testkit.Equal(t, string(result.Files[0].Content), string(want), "basic model must match golden")
	})
}
