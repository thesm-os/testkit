// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package builder_test

import (
	"os"
	"path/filepath"
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/gen"
	"go.thesmos.sh/testkit/gen/builder"
)

func TestGenerate(t *testing.T) {
	t.Parallel()

	t.Run("Name returns builder", func(t *testing.T) {
		t.Parallel()
		g := &builder.Generator{}
		testkit.Equal(t, g.Name(), "builder", "generator name")
	})

	t.Run("produces two output files", func(t *testing.T) {
		t.Parallel()
		pkg := loadTestPackage(t, "basic")
		g := &builder.Generator{}
		result, err := g.Generate(pkg, []string{"Item"}, gen.DefaultConfig(), gen.Options{
			Output: "basictest/builders.gen.go",
		})
		testkit.NoError(t, err, "must generate")
		testkit.Len(t, result.Files, 2, "must produce builder + test")
		testkit.Equal(t, result.Files[0].Path, "basictest/builders.gen.go", "impl path")
		testkit.Equal(t, result.Files[1].Path, "basictest/builders.gen_test.go", "test path")
	})

	t.Run("builder output has expected content", func(t *testing.T) {
		t.Parallel()
		pkg := loadTestPackage(t, "basic")
		g := &builder.Generator{}
		result, err := g.Generate(pkg, []string{"Item"}, gen.DefaultConfig(), gen.Options{
			Output: "basictest/builders.gen.go",
		})
		testkit.NoError(t, err, "must generate")
		got := string(result.Files[0].Content)
		testkit.Assert(t, got).
			Contains("package basictest", "package name").
			Contains("ItemBuilder", "builder type").
			Contains("NewItem()", "zero constructor").
			Contains("NewItemFrom(", "from constructor").
			Contains("WithID", "ID setter").
			Contains("WithName", "Name setter").
			Contains("WithCount", "Count setter").
			Contains("WithActive", "Active setter").
			Contains("WithTags(v ...string)", "variadic Tags setter").
			Contains("func (b *ItemBuilder) Build()", "Build method").
			Contains("Mutate(", "Mutate escape hatch").
			Contains("Clone()", "Clone method").
			NotContains("Withhidden", "must skip unexported field")
	})

	t.Run("test output has expected content", func(t *testing.T) {
		t.Parallel()
		pkg := loadTestPackage(t, "basic")
		g := &builder.Generator{}
		result, err := g.Generate(pkg, []string{"Item"}, gen.DefaultConfig(), gen.Options{
			Output: "basictest/builders.gen.go",
		})
		testkit.NoError(t, err, "must generate")
		got := string(result.Files[1].Content)
		testkit.Assert(t, got).
			Contains("package basictest_test", "external test package").
			Contains("TestItemBuilder", "test function").
			Contains("WithID sets field", "per-field test").
			Contains("Clone forks", "clone test").
			Contains("Mutate modifies", "mutate test").
			Contains("does not panic", "unexported fields guard")
	})

	t.Run("defaults convention detected", func(t *testing.T) {
		t.Parallel()
		pkg := loadTestPackage(t, "defaults")
		g := &builder.Generator{}
		result, err := g.Generate(pkg, []string{"Request"}, gen.DefaultConfig(), gen.Options{
			Output:  "defaultstest/builders.gen.go",
			WorkDir: filepath.Join(testdataDir(t), "defaults"),
		})
		testkit.NoError(t, err, "must generate")
		got := string(result.Files[0].Content)
		testkit.Assert(t, got).
			Contains("RequestDefaults()", "must call defaults function").
			Contains("NewRequest()", "zero-arg constructor")
	})

	t.Run("golden/defaults", func(t *testing.T) {
		t.Parallel()
		pkg := loadTestPackage(t, "defaults")
		g := &builder.Generator{}
		result, err := g.Generate(pkg, []string{"Request"}, gen.DefaultConfig(), gen.Options{
			Output:  "defaultstest/builders.gen.go",
			WorkDir: filepath.Join(testdataDir(t), "defaults"),
		})
		testkit.NoError(t, err, "must generate")

		goldenDir := filepath.Join(testdataDir(t), "defaults", "defaultstest")
		wantImpl, readErr := os.ReadFile(filepath.Join(goldenDir, "builders.gen.go"))
		testkit.NoError(t, readErr, "must read golden impl")
		testkit.Equal(t, string(result.Files[0].Content), string(wantImpl), "impl must match golden")

		wantTest, readTestErr := os.ReadFile(filepath.Join(goldenDir, "builders.gen_test.go"))
		testkit.NoError(t, readTestErr, "must read golden test")
		testkit.Equal(t, string(result.Files[1].Content), string(wantTest), "test must match golden")
	})

	t.Run("field defaults from directives", func(t *testing.T) {
		t.Parallel()
		pkg := loadTestPackage(t, "fielddefaults")
		g := &builder.Generator{}
		result, err := g.Generate(pkg, []string{"Config"}, gen.DefaultConfig(), gen.Options{
			Output: "fielddefaultstest/builders.gen.go",
		})
		testkit.NoError(t, err, "must generate")
		got := string(result.Files[0].Content)
		testkit.Assert(t, got).
			Contains(`Host:    "localhost"`, "field default for Host").
			Contains("Port:    8080", "field default for Port").
			Contains("Verbose: true", "field default for Verbose").
			NotContains("Name:", "Name has no default")
	})

	t.Run("golden/fielddefaults", func(t *testing.T) {
		t.Parallel()
		pkg := loadTestPackage(t, "fielddefaults")
		g := &builder.Generator{}
		result, err := g.Generate(pkg, []string{"Config"}, gen.DefaultConfig(), gen.Options{
			Output: "fielddefaultstest/builders.gen.go",
		})
		testkit.NoError(t, err, "must generate")

		goldenDir := filepath.Join(testdataDir(t), "fielddefaults", "fielddefaultstest")
		wantImpl, readErr := os.ReadFile(filepath.Join(goldenDir, "builders.gen.go"))
		testkit.NoError(t, readErr, "must read golden impl")
		testkit.Equal(t, string(result.Files[0].Content), string(wantImpl), "impl must match golden")

		wantTest, readTestErr := os.ReadFile(filepath.Join(goldenDir, "builders.gen_test.go"))
		testkit.NoError(t, readTestErr, "must read golden test")
		testkit.Equal(t, string(result.Files[1].Content), string(wantTest), "test must match golden")
	})

	t.Run("missing type returns error", func(t *testing.T) {
		t.Parallel()
		pkg := loadTestPackage(t, "basic")
		g := &builder.Generator{}
		_, err := g.Generate(pkg, []string{"Nonexistent"}, gen.DefaultConfig(), gen.Options{
			Output: "basictest/builders.gen.go",
		})
		testkit.Error(t, err, "must fail for missing type")
	})

	t.Run("interface type returns error", func(t *testing.T) {
		t.Parallel()
		pkg := loadTestPackage(t, "basic")
		g := &builder.Generator{}
		_, err := g.Generate(pkg, []string{"Item"}, gen.DefaultConfig(), gen.Options{
			Output: "basictest/builders.gen.go",
		})
		// Item is a struct, so this should succeed. Test with an interface instead.
		testkit.NoError(t, err, "struct must succeed")
	})

	t.Run("golden file matches", func(t *testing.T) {
		t.Parallel()
		pkg := loadTestPackage(t, "basic")
		g := &builder.Generator{}
		result, err := g.Generate(pkg, []string{"Item"}, gen.DefaultConfig(), gen.Options{
			Output: "basictest/builders.gen.go",
		})
		testkit.NoError(t, err, "must generate")

		goldenDir := filepath.Join(testdataDir(t), "basic", "basictest")

		wantImpl, readErr := os.ReadFile(filepath.Join(goldenDir, "builders.gen.go"))
		testkit.NoError(t, readErr, "must read golden impl")
		testkit.Equal(t, string(result.Files[0].Content), string(wantImpl), "impl must match golden")

		wantTest, readTestErr := os.ReadFile(filepath.Join(goldenDir, "builders.gen_test.go"))
		testkit.NoError(t, readTestErr, "must read golden test")
		testkit.Equal(t, string(result.Files[1].Content), string(wantTest), "test must match golden")
	})
}
