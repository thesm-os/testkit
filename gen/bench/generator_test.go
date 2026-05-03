// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package bench_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/gen"
	"go.thesmos.sh/testkit/gen/bench"
)

// suiteTestdataDir returns the path to suite's testdata, used as the
// source interface packages for bench generation.
func suiteTestdataDir(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot determine test file location")
	}
	return filepath.Join(filepath.Dir(file), "..", "suite", "testdata")
}

// benchTestdataDir returns the path to bench's own testdata, where
// generated golden files and bench tests live.
func benchTestdataDir(t *testing.T) string {
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
	dir := filepath.Join(suiteTestdataDir(t), subdir)
	pkg, err := loader.Load(".", dir)
	testkit.NoError(t, err, "must load package")
	return pkg
}

func TestGenerate(t *testing.T) {
	t.Parallel()

	t.Run("Name returns bench", func(t *testing.T) {
		t.Parallel()
		g := &bench.Generator{}
		testkit.Equal(t, g.Name(), "bench", "generator name")
	})

	t.Run("produces one output file", func(t *testing.T) {
		t.Parallel()
		pkg := loadTestPackage(t, "basic")
		g := &bench.Generator{}
		result, err := g.Generate(pkg, []string{"Store"}, gen.DefaultConfig(), gen.Options{
			Output: "storetest/store_bench.gen.go",
		})
		testkit.NoError(t, err, "must generate")
		testkit.Len(t, result.Files, 1, "must produce one bench file")
		testkit.Equal(t, result.Files[0].Path, "storetest/store_bench.gen.go", "output path")
	})

	t.Run("renders basic Store", func(t *testing.T) {
		t.Parallel()
		pkg := loadTestPackage(t, "basic")
		g := &bench.Generator{}
		result, err := g.Generate(pkg, []string{"Store"}, gen.DefaultConfig(), gen.Options{
			Output: "storetest/store_bench.gen.go",
		})
		testkit.NoError(t, err, "must generate")
		content := string(result.Files[0].Content)
		testkit.Assert(t, content).Contains("BenchmarkStoreContract", "must have entry point")
		testkit.Assert(t, content).Contains("benchStoreGet", "must have Reader-shaped method")
		testkit.Assert(t, content).Contains("BenchReaderContext", "must use bench reader context for Get")
		testkit.Assert(t, content).Contains("hot-path", "must have default hot-path benchmark")
		testkit.Assert(t, content).Contains("StoreBenchOption", "must have option type")
		testkit.Assert(t, content).Contains("StoreBenchOnGet", "must have plug-in option for Get")
	})

	goldens := []struct {
		name     string
		dir      string
		typeName string
		output   string
	}{
		{"basic", "basic", "Store", "storetest/store_bench.gen.go"},
		{"allshapes", "allshapes", "Service", "servicetest/service_bench.gen.go"},
	}

	for _, fx := range goldens {
		t.Run("golden/"+fx.name, func(t *testing.T) {
			t.Parallel()
			pkg := loadTestPackage(t, fx.dir)
			g := &bench.Generator{}
			result, err := g.Generate(pkg, []string{fx.typeName}, gen.DefaultConfig(), gen.Options{
				Output: fx.output,
			})
			testkit.NoError(t, err, "must generate "+fx.name)

			goldenFile := filepath.Join(benchTestdataDir(t), fx.dir, filepath.Dir(fx.output), filepath.Base(fx.output))
			want, err := os.ReadFile(goldenFile)
			testkit.NoError(t, err, "must read golden file for "+fx.name)
			testkit.Equal(t, string(result.Files[0].Content), string(want), fx.name+" bench must match golden")
		})
	}
}
