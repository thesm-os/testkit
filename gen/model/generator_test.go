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

	t.Run("renders allshapes Service", func(t *testing.T) {
		t.Parallel()
		pkg := loadTestPackage(t, "allshapes")
		g := &model.Generator{}
		result, err := g.Generate(pkg, []string{"Service"}, gen.DefaultConfig(), gen.Options{
			Output: "servicetest/service_model.gen.go",
		})
		testkit.NoError(t, err, "must generate")
		content := string(result.Files[0].Content)
		testkit.Assert(t, content).Contains("action.Pure", "must emit Pure action")
		testkit.Assert(t, content).Contains("action.Predicate", "must emit Predicate action")
		testkit.Assert(t, content).Contains("action.Stream", "must emit Stream action")
		testkit.Assert(t, content).Contains("action.Lifecycle", "must emit Lifecycle action")
		testkit.Assert(t, content).Contains("action.Deleter", "must emit Deleter action")
		testkit.Assert(t, content).Contains("ServiceModelReference", "must require reference (non-refmap)")
		testkit.Assert(t, content).NotContains("refmap.NewMapStore", "must not synthesize refmap for non-pure-CRUD")
	})

	t.Run("renders noncrud Closer", func(t *testing.T) {
		t.Parallel()
		pkg := loadTestPackage(t, "noncrud")
		g := &model.Generator{}
		result, err := g.Generate(pkg, []string{"Closer"}, gen.DefaultConfig(), gen.Options{
			Output: "closertest/closer_model.gen.go",
		})
		testkit.NoError(t, err, "must generate")
		content := string(result.Files[0].Content)
		testkit.Assert(t, content).Contains("action.Lifecycle", "must emit Lifecycle action")
		testkit.Assert(t, content).NotContains("refmap.NewMapStore", "must not synthesize refmap for non-CRUD")
		testkit.Assert(t, content).Contains("CloserModelReference", "must have reference option")
		testkit.Assert(t, content).Contains("CRUD:          no", "must report non-CRUD")
	})

	t.Run("renders unknown Processor", func(t *testing.T) {
		t.Parallel()
		pkg := loadTestPackage(t, "unknown")
		g := &model.Generator{}
		result, err := g.Generate(pkg, []string{"Processor"}, gen.DefaultConfig(), gen.Options{
			Output: "processortest/processor_model.gen.go",
		})
		testkit.NoError(t, err, "must generate")
		content := string(result.Files[0].Content)
		testkit.Assert(t, content).Contains("Skipped:", "must have Skipped line")
		testkit.Assert(t, content).Contains("Process(Unknown)", "must report Process as Unknown")
	})

	t.Run("renders keyfield Store", func(t *testing.T) {
		t.Parallel()
		pkg := loadTestPackage(t, "keyfield")
		g := &model.Generator{}
		result, err := g.Generate(pkg, []string{"Store"}, gen.DefaultConfig(), gen.Options{
			Output: "storetest/store_model.gen.go",
		})
		testkit.NoError(t, err, "must generate")
		content := string(result.Files[0].Content)
		testkit.Assert(t, content).Contains("Key field:     Key", "must use directive keyfield")
		testkit.Assert(t, content).Contains(`"Key": keyGen.AsAny()`, "must override Key in MakeCustom")
		testkit.Assert(t, content).Contains("return v.Key", "must extract Key in refmap")
	})

	t.Run("renders generic ItemRepository (alias/monomorphic)", func(t *testing.T) {
		t.Parallel()
		pkg := loadTestPackage(t, "generic")
		g := &model.Generator{}
		result, err := g.Generate(pkg, []string{"ItemRepository"}, gen.DefaultConfig(), gen.Options{
			Output: "repotest/repo_model.gen.go",
		})
		testkit.NoError(t, err, "must generate")
		content := string(result.Files[0].Content)
		testkit.Assert(t, content).Contains("AssertItemRepositoryModel", "must use alias name")
		testkit.Assert(t, content).Contains("generic.ItemRepository", "must use qualified alias type")
		testkit.Assert(t, content).Contains("action.Deleter", "must pick up //testkit:deleter via origin")
		testkit.Assert(t, content).Contains("generic.ErrNotFound", "must pick up sentinel via origin")
		testkit.Assert(t, content).Contains("AUTO-DELETE-RETURNS-NOT-FOUND", "must derive delete law")
	})

	t.Run("renders generic Repository (parameterized)", func(t *testing.T) {
		t.Parallel()
		pkg := loadTestPackage(t, "generic")
		g := &model.Generator{}
		result, err := g.Generate(pkg, []string{"Repository"}, gen.DefaultConfig(), gen.Options{
			Output: "repotest/repo_generic_model.gen.go",
		})
		testkit.NoError(t, err, "must generate")
		content := string(result.Files[0].Content)
		testkit.Assert(t, content).
			Contains("[K comparable, V any]", "must have type param decl")
		testkit.Assert(t, content).
			Contains("generic.Repository[K, V]", "must use parameterized type")
		testkit.Assert(t, content).
			Contains("rapid.Make[V]()", "must use rapid.Make for generic V")
		testkit.Assert(t, content).
			Contains("RepositoryModelKeyGen", "must have keyGen option")
		testkit.Assert(t, content).
			Contains("RepositoryModelKeyFunc", "must have keyFunc option")
		testkit.Assert(t, content).
			Contains("RepositoryModelSentinel", "must have sentinel option")
		testkit.Assert(t, content).
			NotContains("rapid.MakeCustom", "must not use MakeCustom for generic")
		testkit.Assert(t, content).
			NotContains("reflect.TypeOf", "must not use reflect for generic")
	})

	t.Run("renders richstruct Store", func(t *testing.T) {
		t.Parallel()
		pkg := loadTestPackage(t, "richstruct")
		g := &model.Generator{}
		result, err := g.Generate(pkg, []string{"Store"}, gen.DefaultConfig(), gen.Options{
			Output: "storetest/store_model.gen.go",
		})
		testkit.NoError(t, err, "must generate")
		content := string(result.Files[0].Content)
		testkit.Assert(t, content).Contains("rapid.MakeCustom", "must use MakeCustom")
		testkit.Assert(t, content).Contains(`"ID": keyGen.AsAny()`, "must override keyfield")
		testkit.Assert(t, content).Contains("refmap.NewMapStore", "must synthesize refmap (pure CRUD)")
	})

	t.Run("renders multisentinel Vault", func(t *testing.T) {
		t.Parallel()
		pkg := loadTestPackage(t, "multisentinel")
		g := &model.Generator{}
		result, err := g.Generate(pkg, []string{"Vault"}, gen.DefaultConfig(), gen.Options{
			Output: "vaulttest/vault_model.gen.go",
		})
		testkit.NoError(t, err, "must generate")
		content := string(result.Files[0].Content)
		testkit.Assert(t, content).Contains("multisentinel.ErrNotFound", "must pick first sentinel")
		testkit.Assert(t, content).NotContains("multisentinel.ErrSealed", "must not use second sentinel")
		testkit.Assert(t, content).Contains("refmap.NewMapStore", "must synthesize refmap (pure CRUD)")
		testkit.Assert(t, content).Contains(`Ref:           auto (refmap.MapStore)`, "must show auto ref")
	})

	t.Run("renders allshapes ref header", func(t *testing.T) {
		t.Parallel()
		pkg := loadTestPackage(t, "allshapes")
		g := &model.Generator{}
		result, err := g.Generate(pkg, []string{"Service"}, gen.DefaultConfig(), gen.Options{
			Output: "servicetest/service_model.gen.go",
		})
		testkit.NoError(t, err, "must generate")
		content := string(result.Files[0].Content)
		testkit.Assert(t, content).
			Contains("supply via ServiceModelReference", "must explain ref unavailable")
		testkit.Assert(t, content).Contains("Close", "must list blocking methods")
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

	t.Run("golden/allshapes", func(t *testing.T) {
		t.Parallel()
		pkg := loadTestPackage(t, "allshapes")
		g := &model.Generator{}
		result, err := g.Generate(pkg, []string{"Service"}, gen.DefaultConfig(), gen.Options{
			Output: "servicetest/service_model.gen.go",
		})
		testkit.NoError(t, err, "must generate")

		goldenFile := filepath.Join(testdataDir(t), "allshapes", "servicetest", "service_model.gen.go")
		want, err := os.ReadFile(goldenFile)
		testkit.NoError(t, err, "must read golden file")
		testkit.Equal(t, string(result.Files[0].Content), string(want), "allshapes model must match golden")
	})

	t.Run("golden/noncrud", func(t *testing.T) {
		t.Parallel()
		pkg := loadTestPackage(t, "noncrud")
		g := &model.Generator{}
		result, err := g.Generate(pkg, []string{"Closer"}, gen.DefaultConfig(), gen.Options{
			Output: "closertest/closer_model.gen.go",
		})
		testkit.NoError(t, err, "must generate")

		goldenFile := filepath.Join(testdataDir(t), "noncrud", "closertest", "closer_model.gen.go")
		want, err := os.ReadFile(goldenFile)
		testkit.NoError(t, err, "must read golden file")
		testkit.Equal(t, string(result.Files[0].Content), string(want), "noncrud model must match golden")
	})

	t.Run("golden/unknown", func(t *testing.T) {
		t.Parallel()
		pkg := loadTestPackage(t, "unknown")
		g := &model.Generator{}
		result, err := g.Generate(pkg, []string{"Processor"}, gen.DefaultConfig(), gen.Options{
			Output: "processortest/processor_model.gen.go",
		})
		testkit.NoError(t, err, "must generate")

		goldenFile := filepath.Join(testdataDir(t), "unknown", "processortest", "processor_model.gen.go")
		want, err := os.ReadFile(goldenFile)
		testkit.NoError(t, err, "must read golden file")
		testkit.Equal(t, string(result.Files[0].Content), string(want), "unknown model must match golden")
	})

	t.Run("golden/keyfield", func(t *testing.T) {
		t.Parallel()
		pkg := loadTestPackage(t, "keyfield")
		g := &model.Generator{}
		result, err := g.Generate(pkg, []string{"Store"}, gen.DefaultConfig(), gen.Options{
			Output: "storetest/store_model.gen.go",
		})
		testkit.NoError(t, err, "must generate")

		goldenFile := filepath.Join(testdataDir(t), "keyfield", "storetest", "store_model.gen.go")
		want, err := os.ReadFile(goldenFile)
		testkit.NoError(t, err, "must read golden file")
		testkit.Equal(t, string(result.Files[0].Content), string(want), "keyfield model must match golden")
	})

	t.Run("golden/multisentinel", func(t *testing.T) {
		t.Parallel()
		pkg := loadTestPackage(t, "multisentinel")
		g := &model.Generator{}
		result, err := g.Generate(pkg, []string{"Vault"}, gen.DefaultConfig(), gen.Options{
			Output: "vaulttest/vault_model.gen.go",
		})
		testkit.NoError(t, err, "must generate")

		goldenFile := filepath.Join(testdataDir(t), "multisentinel", "vaulttest", "vault_model.gen.go")
		want, err := os.ReadFile(goldenFile)
		testkit.NoError(t, err, "must read golden file")
		testkit.Equal(t, string(result.Files[0].Content), string(want), "multisentinel model must match golden")
	})

	t.Run("golden/richstruct", func(t *testing.T) {
		t.Parallel()
		pkg := loadTestPackage(t, "richstruct")
		g := &model.Generator{}
		result, err := g.Generate(pkg, []string{"Store"}, gen.DefaultConfig(), gen.Options{
			Output: "storetest/store_model.gen.go",
		})
		testkit.NoError(t, err, "must generate")

		goldenFile := filepath.Join(testdataDir(t), "richstruct", "storetest", "store_model.gen.go")
		want, err := os.ReadFile(goldenFile)
		testkit.NoError(t, err, "must read golden file")
		testkit.Equal(t, string(result.Files[0].Content), string(want), "richstruct model must match golden")
	})

	t.Run("golden/generic-alias", func(t *testing.T) {
		t.Parallel()
		pkg := loadTestPackage(t, "generic")
		g := &model.Generator{}
		result, err := g.Generate(pkg, []string{"ItemRepository"}, gen.DefaultConfig(), gen.Options{
			Output: "repotest/repo_model.gen.go",
		})
		testkit.NoError(t, err, "must generate")

		goldenFile := filepath.Join(testdataDir(t), "generic", "repotest", "repo_model.gen.go")
		want, err := os.ReadFile(goldenFile)
		testkit.NoError(t, err, "must read golden file")
		testkit.Equal(t, string(result.Files[0].Content), string(want), "generic alias model must match golden")
	})

	t.Run("golden/generic-parameterized", func(t *testing.T) {
		t.Parallel()
		pkg := loadTestPackage(t, "generic")
		g := &model.Generator{}
		result, err := g.Generate(pkg, []string{"Repository"}, gen.DefaultConfig(), gen.Options{
			Output: "repotest/repo_generic_model.gen.go",
		})
		testkit.NoError(t, err, "must generate")

		goldenFile := filepath.Join(testdataDir(t), "generic", "repotest", "repo_generic_model.gen.go")
		want, err := os.ReadFile(goldenFile)
		testkit.NoError(t, err, "must read golden file")
		testkit.Equal(t, string(result.Files[0].Content), string(want), "generic parameterized model must match golden")
	})
}
