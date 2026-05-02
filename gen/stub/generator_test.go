// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package stub_test

import (
	"os"
	"path/filepath"
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/gen"
	"go.thesmos.sh/testkit/gen/stub"
)

func TestGenerate(t *testing.T) {
	t.Parallel()

	t.Run("templates parse without error", func(t *testing.T) {
		t.Parallel()
		tmpl := gen.NewTemplateSet()
		_, err := tmpl.ParseFS(stub.TemplateFS(), "templates/*.tmpl")
		testkit.NoError(t, err, "all templates must parse")
	})

	t.Run("produces two output files", func(t *testing.T) {
		t.Parallel()
		pkg := loadTestPackage(t, "basic")
		g := &stub.Generator{}
		result, err := g.Generate(pkg, []string{"Store"}, gen.DefaultConfig(), gen.Options{
			Output: "storetest/store_stub.gen.go",
		})
		testkit.NoError(t, err, "must generate")
		testkit.Len(t, result.Files, 2, "must produce stub + test")
		testkit.Equal(t, result.Files[0].Path, "storetest/store_stub.gen.go", "impl path")
		testkit.Equal(t, result.Files[1].Path, "storetest/store_stub.gen_test.go", "test path")
	})

	t.Run("stub output has expected content", func(t *testing.T) {
		t.Parallel()
		pkg := loadTestPackage(t, "basic")
		g := &stub.Generator{}
		result, err := g.Generate(pkg, []string{"Store"}, gen.DefaultConfig(), gen.Options{
			Output: "storetest/store_stub.gen.go",
		})
		testkit.NoError(t, err, "must generate")
		got := string(result.Files[0].Content)
		testkit.Assert(t, got).
			Contains("package storetest", "package name").
			Contains("var _ basic.Store = (*StoreStub)(nil)", "compile check").
			Contains("StoreGetCall", "call type").
			Contains("StoreGetStub", "stub type").
			Contains("NewStoreStub", "constructor").
			Contains("func (s *StoreStub) Get(", "method impl")
	})

	t.Run("test output has expected content", func(t *testing.T) {
		t.Parallel()
		pkg := loadTestPackage(t, "basic")
		g := &stub.Generator{}
		result, err := g.Generate(pkg, []string{"Store"}, gen.DefaultConfig(), gen.Options{
			Output: "storetest/store_stub.gen.go",
		})
		testkit.NoError(t, err, "must generate")
		got := string(result.Files[1].Content)
		testkit.Assert(t, got).
			Contains("package storetest_test", "external test package").
			Contains("TestStoreStub", "test function").
			Contains("Get default returns zero", "subtest")
	})

	t.Run("errors directive produces fault helpers", func(t *testing.T) {
		t.Parallel()
		pkg := loadTestPackage(t, "directives")
		g := &stub.Generator{}
		result, err := g.Generate(pkg, []string{"Store"}, gen.DefaultConfig(), gen.Options{
			Output: "storetest/store_stub.gen.go",
		})
		testkit.NoError(t, err, "must generate")
		got := string(result.Files[0].Content)
		testkit.Assert(t, got).
			Contains("FaultNotFound", "errors directive fault helper").
			Contains("FaultConflict", "errors directive fault helper")
	})

	t.Run("missing type returns error", func(t *testing.T) {
		t.Parallel()
		pkg := loadTestPackage(t, "basic")
		g := &stub.Generator{}
		_, err := g.Generate(pkg, []string{"Nonexistent"}, gen.DefaultConfig(), gen.Options{
			Output: "storetest/store_stub.gen.go",
		})
		testkit.Error(t, err, "must fail for missing type")
	})

	t.Run("struct type returns error", func(t *testing.T) {
		t.Parallel()
		pkg := loadTestPackage(t, "basic")
		g := &stub.Generator{}
		_, err := g.Generate(pkg, []string{"Item"}, gen.DefaultConfig(), gen.Options{
			Output: "storetest/store_stub.gen.go",
		})
		testkit.Error(t, err, "must fail for struct")
	})

	t.Run("Name returns stub", func(t *testing.T) {
		t.Parallel()
		g := &stub.Generator{}
		testkit.Equal(t, g.Name(), "stub", "generator name")
	})

	t.Run("output matches golden files", func(t *testing.T) {
		t.Parallel()
		pkg := loadTestPackage(t, "basic")
		g := &stub.Generator{}
		result, err := g.Generate(pkg, []string{"Store"}, gen.DefaultConfig(), gen.Options{
			Output: "storetest/store_stub.gen.go",
		})
		testkit.NoError(t, err, "must generate")

		goldenDir := filepath.Join(testdataDir(t), "basic", "storetest")

		wantImpl, err := os.ReadFile(filepath.Join(goldenDir, "store_stub.gen.go"))
		testkit.NoError(t, err, "must read golden impl")
		testkit.Equal(t, string(result.Files[0].Content), string(wantImpl), "impl must match golden")

		wantTest, err := os.ReadFile(filepath.Join(goldenDir, "store_stub.gen_test.go"))
		testkit.NoError(t, err, "must read golden test")
		testkit.Equal(t, string(result.Files[1].Content), string(wantTest), "test must match golden")
	})
}
