// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package bench_test

import (
	"strings"
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/generator"
	"go.thesmos.sh/testkit/generator/bench"
)

func loadFixture(t *testing.T, name string) *generator.Package {
	t.Helper()
	pkg, err := generator.NewLoader().Load("./../testdata/"+name, "")
	testkit.NoError(t, err, "Load testdata/"+name)
	return pkg
}

func TestGenerator(t *testing.T) {
	t.Parallel()

	t.Run("Name returns subcommand", func(t *testing.T) {
		t.Parallel()
		testkit.Equal(t, (&bench.Generator{}).Name(), bench.Name, "Name")
	})

	t.Run("emits one file at the configured Output path", func(t *testing.T) {
		t.Parallel()
		pkg := loadFixture(t, "basic")
		res, err := (&bench.Generator{}).Generate(pkg, []string{"Store"},
			generator.DefaultConfig(), generator.Options{
				Output: "storetest/store_bench.gen.go",
			})
		testkit.NoError(t, err, "Generate")
		testkit.Len(t, res.Files, 1, "single output file")
		testkit.Equal(t, res.Files[0].Path, "storetest/store_bench.gen.go", "configured path")
	})

	t.Run("driver function follows Benchmark<Iface>Contract naming", func(t *testing.T) {
		t.Parallel()
		pkg := loadFixture(t, "basic")
		res, err := (&bench.Generator{}).Generate(pkg, []string{"Store"},
			generator.DefaultConfig(), generator.Options{
				Output: "storetest/store_bench.gen.go",
			})
		testkit.NoError(t, err, "Generate")
		out := string(res.Files[0].Content)
		testkit.Assert(t, out).
			Contains("func BenchmarkStoreContract(b *testing.B, factory func() basic.Store, opts ...StoreBenchOption)",
				"driver signature").
			Contains("type StoreBenchOption func(*storeBenchConfig)",
				"option type alias").
			Contains("func StoreBenchPrePopulate(",
				"prePopulate option constructor").
			Contains("func StoreBenchCustom(",
				"custom escape-hatch option")
	})

	t.Run("emits one On<Method> option per non-skip method", func(t *testing.T) {
		t.Parallel()
		pkg := loadFixture(t, "basic")
		res, err := (&bench.Generator{}).Generate(pkg, []string{"Store"},
			generator.DefaultConfig(), generator.Options{
				Output: "storetest/store_bench.gen.go",
			})
		testkit.NoError(t, err, "Generate")
		out := string(res.Files[0].Content)
		testkit.Assert(t, out).
			Contains("func StoreBenchOnGet(", "Get option").
			Contains("func StoreBenchOnPut(", "Put option")
	})

	t.Run("per-method helpers always emit HotPath + ConcurrentThroughput", func(t *testing.T) {
		t.Parallel()
		pkg := loadFixture(t, "basic")
		res, err := (&bench.Generator{}).Generate(pkg, []string{"Store"},
			generator.DefaultConfig(), generator.Options{
				Output: "storetest/store_bench.gen.go",
			})
		testkit.NoError(t, err, "Generate")
		out := string(res.Files[0].Content)
		testkit.Assert(t, out).
			Contains("bench.ReaderHotPath", "HotPath emitted").
			Contains("bench.ReaderConcurrentThroughput", "ConcurrentThroughput emitted")
	})

	t.Run("opt-in budget gates omitted when no //testkit:allocs / //testkit:latency", func(t *testing.T) {
		t.Parallel()
		pkg := loadFixture(t, "basic")
		res, err := (&bench.Generator{}).Generate(pkg, []string{"Store"},
			generator.DefaultConfig(), generator.Options{
				Output: "storetest/store_bench.gen.go",
			})
		testkit.NoError(t, err, "Generate")
		out := string(res.Files[0].Content)
		// Store has no //testkit:allocs / //testkit:latency directives
		// → the AllocsWithin / LatencyWithin calls must NOT appear.
		testkit.False(t, strings.Contains(out, "ReaderAllocsWithin"),
			"AllocsWithin omitted when no allocs directive")
		testkit.False(t, strings.Contains(out, "ReaderLatencyWithin"),
			"LatencyWithin omitted when no latency directive")
	})

	t.Run("opt-in budget gates emitted with //testkit:allocs + //testkit:latency", func(t *testing.T) {
		t.Parallel()
		pkg := loadFixture(t, "basic")
		res, err := (&bench.Generator{}).Generate(pkg, []string{"Perf"},
			generator.DefaultConfig(), generator.Options{
				Output: "perftest/perf_bench.gen.go",
			})
		testkit.NoError(t, err, "Generate")
		out := string(res.Files[0].Content)
		// Perf.Hot carries both directives — the gates must emit.
		testkit.Assert(t, out).
			Contains("bench.ReaderAllocsWithin", "AllocsWithin emitted").
			Contains("bench.ReaderLatencyWithin", "LatencyWithin emitted").
			Contains("time.Duration(", "latency rendered as time.Duration").
			Contains("/* 100us */", "latency carries raw-arg comment")
	})

	t.Run("integration-only methods skipped", func(t *testing.T) {
		t.Parallel()
		pkg, err := generator.NewLoader().Load("./../testdata/interfaces", "")
		testkit.NoError(t, err, "Load interfaces fixture")
		res, err := (&bench.Generator{}).Generate(pkg, []string{"Directives"},
			generator.DefaultConfig(), generator.Options{
				Output: "directivestest/directives_bench.gen.go",
			})
		testkit.NoError(t, err, "Generate")
		out := string(res.Files[0].Content)
		// Close on Directives is //testkit:integration-only — no helper emitted.
		testkit.False(t, strings.Contains(out, "benchDirectivesClose"),
			"integration-only method has no helper")
	})

	t.Run("AllShapes covers every shape that classifies", func(t *testing.T) {
		t.Parallel()
		pkg, err := generator.NewLoader().Load("./../testdata/interfaces", "")
		testkit.NoError(t, err, "Load interfaces fixture")
		res, err := (&bench.Generator{}).Generate(pkg, []string{"AllShapes"},
			generator.DefaultConfig(), generator.Options{
				Output: "allshapestest/allshapes_bench.gen.go",
			})
		testkit.NoError(t, err, "Generate AllShapes")
		out := string(res.Files[0].Content)
		// Spot-check representative shapes.
		testkit.Assert(t, out).
			Contains("bench.ReaderHotPath", "Reader primitive emitted").
			Contains("bench.WriterHotPath", "Writer primitive emitted").
			Contains("bench.MutatorHotPath", "Mutator primitive emitted").
			Contains("bench.LifecycleHotPath", "Lifecycle primitive emitted")
	})

	t.Run("non-interface arg surfaces a hard error", func(t *testing.T) {
		t.Parallel()
		pkg := loadFixture(t, "basic")
		_, err := (&bench.Generator{}).Generate(pkg, []string{"Status"},
			generator.DefaultConfig(), generator.Options{
				Output: "x.gen.go",
			})
		testkit.True(t, err != nil, "non-interface rejected")
	})

	t.Run("generic interface emits concrete instantiation", func(t *testing.T) {
		t.Parallel()
		pkg := loadFixture(t, "generics")
		res, err := (&bench.Generator{}).Generate(pkg, []string{"Holder"},
			generator.DefaultConfig(), generator.Options{
				Output: "genericstest/holder_bench.gen.go",
			})
		testkit.NoError(t, err, "Generate Holder")
		out := string(res.Files[0].Content)
		testkit.Assert(t, out).
			Contains("Holder", "references Holder").
			Contains("[string]", "concrete type arg for V any → string")
	})
}

func analyzeAndProject(t *testing.T, fixture, iface string) *bench.Data {
	t.Helper()
	pkg := loadFixture(t, fixture)
	d, err := bench.Analyze(pkg, []string{iface}, generator.DefaultConfig(), generator.Options{
		Output: "test/out.gen.go",
	})
	testkit.NoError(t, err, "Analyze")
	testkit.NoError(t, bench.Enrich(d, pkg), "Enrich")
	testkit.NoError(t, bench.Project(d, pkg), "Project")
	return d
}

func TestAnalyze(t *testing.T) {
	t.Parallel()

	t.Run("error from spec.Analyze propagates", func(t *testing.T) {
		t.Parallel()
		pkg := loadFixture(t, "basic")
		_, err := bench.Analyze(pkg, []string{"DoesNotExist"},
			generator.DefaultConfig(), generator.Options{Output: "x.gen.go"})
		testkit.True(t, err != nil, "nonexistent interface rejected")
	})

	t.Run("DriverName follows Benchmark<Iface>Contract pattern", func(t *testing.T) {
		t.Parallel()
		d := analyzeAndProject(t, "basic", "Store")
		testkit.Equal(t, d.DriverName, "BenchmarkStoreContract", "DriverName")
	})

	t.Run("LowerIfaceName lowercases first rune", func(t *testing.T) {
		t.Parallel()
		d := analyzeAndProject(t, "basic", "Store")
		testkit.Equal(t, d.LowerIfaceName, "store", "LowerIfaceName")
	})

	t.Run("ConfigTypeName follows pattern", func(t *testing.T) {
		t.Parallel()
		d := analyzeAndProject(t, "basic", "Store")
		testkit.Equal(t, d.ConfigTypeName, "storeBenchConfig", "ConfigTypeName")
	})

	t.Run("OptionTypeName follows pattern", func(t *testing.T) {
		t.Parallel()
		d := analyzeAndProject(t, "basic", "Store")
		testkit.Equal(t, d.OptionTypeName, "StoreBenchOption", "OptionTypeName")
	})
}

func methodByName(t *testing.T, d *bench.Data, name string) *bench.MethodView {
	t.Helper()
	for i := range d.Methods {
		if d.Methods[i].Name == name {
			return &d.Methods[i]
		}
	}
	t.Fatalf("method %q not found", name)
	return nil
}

func TestMethodView(t *testing.T) {
	t.Parallel()

	t.Run("HelperName follows bench<Iface><Method> pattern", func(t *testing.T) {
		t.Parallel()
		d := analyzeAndProject(t, "basic", "Store")
		get := methodByName(t, d, "Get")
		testkit.Equal(t, get.HelperName(), "benchStoreGet", "HelperName")
	})

	t.Run("OnOptionName follows <Iface>BenchOn<Method> pattern", func(t *testing.T) {
		t.Parallel()
		d := analyzeAndProject(t, "basic", "Store")
		get := methodByName(t, d, "Get")
		testkit.Equal(t, get.OnOptionName(), "StoreBenchOnGet", "OnOptionName")
	})

	t.Run("OnFieldName follows on<Method> pattern", func(t *testing.T) {
		t.Parallel()
		d := analyzeAndProject(t, "basic", "Store")
		get := methodByName(t, d, "Get")
		testkit.Equal(t, get.OnFieldName(), "onGet", "OnFieldName")
	})

	t.Run("HasAllocsBudget false without directive", func(t *testing.T) {
		t.Parallel()
		d := analyzeAndProject(t, "basic", "Store")
		get := methodByName(t, d, "Get")
		testkit.False(t, get.HasAllocsBudget(), "Store.Get has no allocs directive")
	})

	t.Run("HasLatencyBudget false without directive", func(t *testing.T) {
		t.Parallel()
		d := analyzeAndProject(t, "basic", "Store")
		get := methodByName(t, d, "Get")
		testkit.False(t, get.HasLatencyBudget(), "Store.Get has no latency directive")
	})

	t.Run("OnMethodBenchType renders typed primitive for known shape", func(t *testing.T) {
		t.Parallel()
		d := analyzeAndProject(t, "basic", "Store")
		get := methodByName(t, d, "Get")
		got := get.OnMethodBenchType("basic.Store", d.Spec.Tracker)
		testkit.Assert(t, got).
			Contains("bench.Reader", "Reader shape").
			Contains("basic.Store", "iface qualified")
	})

	t.Run("HasShapePrimitive true for typed shapes", func(t *testing.T) {
		t.Parallel()
		d := analyzeAndProject(t, "basic", "Store")
		get := methodByName(t, d, "Get")
		testkit.True(t, get.HasShapePrimitive(), "Reader is a typed shape")
	})
}
