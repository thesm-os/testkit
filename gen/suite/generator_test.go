// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package suite_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/gen"
	"go.thesmos.sh/testkit/gen/suite"
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

	t.Run("Name returns suite", func(t *testing.T) {
		t.Parallel()
		g := &suite.Generator{}
		testkit.Equal(t, g.Name(), "suite", "generator name")
	})

	t.Run("produces one output file", func(t *testing.T) {
		t.Parallel()
		pkg := loadTestPackage(t, "basic")
		g := &suite.Generator{}
		result, err := g.Generate(pkg, []string{"Store"}, gen.DefaultConfig(), gen.Options{
			Output: "storetest/store_spec.gen.go",
		})
		testkit.NoError(t, err, "must generate")
		testkit.Len(t, result.Files, 1, "must produce one spec file")
		testkit.Equal(t, result.Files[0].Path, "storetest/store_spec.gen.go", "output path")
	})

	t.Run("templates parse without error", func(t *testing.T) {
		t.Parallel()
		tmpl := gen.NewTemplateSet()
		_, err := tmpl.ParseFS(suite.TemplateFS(), "templates/*.tmpl")
		testkit.NoError(t, err, "all templates must parse")
	})

	fixtures := []struct {
		name     string
		dir      string
		typeName string
		output   string // explicit output path; empty = use first //go:generate directive
	}{
		{"basic", "basic", "Store", ""},
		{"nocontext", "nocontext", "Cache", ""},
		{"multireturn", "multireturn", "Service", ""},
		{"mixed", "mixed", "Processor", ""},
		{"erroronly", "erroronly", "Closer", ""},
		{"iterators", "iterators", "Scanner", ""},
		{"readers", "readers", "Registry", ""},
		{"writers", "writers", "Store", ""},
		{"allshapes", "allshapes", "Service", ""},
		{"weird/codec", "weird", "Codec", "weirdtest/weird_spec.gen.go"},
		{"weird/scheduler", "weird", "Scheduler", "weirdtest/scheduler_spec.gen.go"},
		{"voidpure", "voidpure", "Stream", ""},
		{"samples", "samples", "Hasher", ""},
	}

	t.Run("samples uses sample builders in generated output", func(t *testing.T) {
		t.Parallel()
		pkg := loadTestPackage(t, "samples")
		g := &suite.Generator{}
		result, err := g.Generate(pkg, []string{"Hasher"}, gen.DefaultConfig(), gen.Options{
			Output: "hashertest/hasher_spec.gen.go",
		})
		testkit.NoError(t, err, "must generate")
		content := string(result.Files[0].Content)
		testkit.Contains(t, content, "samples.SampleDigest(impl)", "source-package sample qualified in Call closure")
		testkit.Contains(t, content, "samples.SampleDigest(s)", "source-package sample qualified in smoke test")
		testkit.Contains(t, content, "TestSampleDigest(impl)", "output-package sample unqualified in Call closure")
		testkit.Contains(t, content, "TestSampleDigest(s)", "output-package sample unqualified in smoke test")
	})

	for _, fx := range fixtures {
		t.Run("golden/"+fx.name, func(t *testing.T) {
			t.Parallel()
			pkg := loadTestPackage(t, fx.dir)
			g := &suite.Generator{}

			outputPath := fx.output
			if outputPath == "" {
				dirs := pkg.GenerateDirectives()
				outputPath = "storetest/store_spec.gen.go"
				if len(dirs) > 0 {
					outputPath = dirs[0].Output
				}
			}

			result, err := g.Generate(pkg, []string{fx.typeName}, gen.DefaultConfig(), gen.Options{
				Output: outputPath,
			})
			testkit.NoError(t, err, "must generate "+fx.name)

			goldenDir := filepath.Join(testdataDir(t), fx.dir, filepath.Dir(outputPath))
			goldenFile := filepath.Join(goldenDir, filepath.Base(outputPath))
			want, err := os.ReadFile(goldenFile)
			testkit.NoError(t, err, "must read golden file for "+fx.name)
			testkit.Equal(t, string(result.Files[0].Content), string(want), fx.name+" must match golden")
		})
	}
}
