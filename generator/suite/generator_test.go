// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package suite_test

import (
	"strings"
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/generator"
	"go.thesmos.sh/testkit/generator/suite"
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
		testkit.Equal(t, (&suite.Generator{}).Name(), suite.Name, "Name")
	})

	t.Run("emits one file at the configured Output path", func(t *testing.T) {
		t.Parallel()
		pkg := loadFixture(t, "basic")
		res, err := (&suite.Generator{}).Generate(pkg, []string{"Store"},
			generator.DefaultConfig(), generator.Options{
				Output: "storetest/store_spec.gen.go",
			})
		testkit.NoError(t, err, "Generate")
		testkit.Len(t, res.Files, 1, "single output file")
		testkit.Equal(t, res.Files[0].Path, "storetest/store_spec.gen.go", "configured path")
	})

	t.Run("driver functions follow Assert<Iface>Contract naming", func(t *testing.T) {
		t.Parallel()
		pkg := loadFixture(t, "basic")
		res, err := (&suite.Generator{}).Generate(pkg, []string{"Store"},
			generator.DefaultConfig(), generator.Options{
				Output: "storetest/store_spec.gen.go",
			})
		testkit.NoError(t, err, "Generate")
		out := string(res.Files[0].Content)
		testkit.Assert(t, out).
			Contains("func AssertStoreContract(t *testing.T, factory StoreFactory, opts ...suite.Option)",
				"single-impl driver accepts ...Option").
			Contains("resolved := suite.ResolveOptions(opts...)",
				"driver folds options once").
			Contains("func AssertStoreContractAcrossImpls(t *testing.T, factories []StoreNamedFactory, opts ...suite.Option)",
				"multi-impl driver takes []NamedFactory + ...Option").
			Contains("type StoreFactory func() basic.Store", "factory alias").
			Contains("type StoreNamedFactory struct", "named factory tuple")
	})

	t.Run("per-method baseline emitted per non-skip method", func(t *testing.T) {
		t.Parallel()
		pkg := loadFixture(t, "basic")
		res, err := (&suite.Generator{}).Generate(pkg, []string{"Store"},
			generator.DefaultConfig(), generator.Options{
				Output: "storetest/store_spec.gen.go",
			})
		testkit.NoError(t, err, "Generate")
		out := string(res.Files[0].Content)
		testkit.Assert(t, out).
			Contains(`t.Run("Get"`, "Get per-method block").
			Contains(`t.Run("Put"`, "Put per-method block").
			Contains(`suite.AssertReaderBaseline[`, "Reader baseline call (smoke folded into runtime)").
			Contains(`suite.AssertWriterBaseline[`, "Writer baseline call (smoke folded into runtime)")
	})

	t.Run("integration-only methods skipped", func(t *testing.T) {
		t.Parallel()
		pkg, err := generator.NewLoader().Load("./../testdata/interfaces", "")
		testkit.NoError(t, err, "Load interfaces fixture")
		res, err := (&suite.Generator{}).Generate(pkg, []string{"Directives"},
			generator.DefaultConfig(), generator.Options{
				Output: "directivestest/directives_spec.gen.go",
			})
		testkit.NoError(t, err, "Generate")
		out := string(res.Files[0].Content)
		testkit.False(t, strings.Contains(out, `t.Run("Close"`),
			"Close is //testkit:integration-only — must be skipped")
	})

	t.Run("non-interface arg surfaces a hard error", func(t *testing.T) {
		t.Parallel()
		pkg := loadFixture(t, "basic")
		_, err := (&suite.Generator{}).Generate(pkg, []string{"Status"},
			generator.DefaultConfig(), generator.Options{
				Output: "x.gen.go",
			})
		testkit.True(t, err != nil, "non-interface rejected")
	})

	t.Run("AllShapes emits per-shape baseline calls", func(t *testing.T) {
		t.Parallel()
		pkg, err := generator.NewLoader().Load("./../testdata/interfaces", "")
		testkit.NoError(t, err, "Load interfaces fixture")
		res, err := (&suite.Generator{}).Generate(pkg, []string{"AllShapes"},
			generator.DefaultConfig(), generator.Options{
				Output: "allshapestest/allshapes_spec.gen.go",
			})
		testkit.NoError(t, err, "Generate AllShapes")
		out := string(res.Files[0].Content)
		// Verify representative shapes get per-method blocks each
		// dispatching to its Assert<Shape>Baseline runtime helper.
		// Smoke runs inside the baseline; no separate t.Run("smoke")
		// appears in the emission.
		testkit.Assert(t, out).
			Contains(`t.Run("Get"`, "Reader per-method block").
			Contains(`t.Run("Put"`, "Writer per-method block").
			Contains(`t.Run("Touch"`, "Mutator per-method block").
			Contains(`t.Run("Init"`, "Lifecycle per-method block").
			Contains(`suite.AssertReaderBaseline[`, "Reader baseline call").
			Contains(`suite.AssertWriterBaseline[`, "Writer baseline call").
			Contains(`suite.AssertMutatorBaseline[`, "Mutator baseline call").
			Contains(`suite.AssertLifecycleBaseline[`, "Lifecycle baseline call")
	})

	t.Run("generic interface emits concrete instantiation", func(t *testing.T) {
		t.Parallel()
		pkg := loadFixture(t, "generics")
		res, err := (&suite.Generator{}).Generate(pkg, []string{"Holder"},
			generator.DefaultConfig(), generator.Options{
				Output: "genericstest/holder_spec.gen.go",
			})
		testkit.NoError(t, err, "Generate Holder")
		out := string(res.Files[0].Content)
		// Generic: driver should reference the concrete instantiation.
		testkit.Assert(t, out).
			Contains("Holder", "references Holder").
			Contains("[string]", "concrete type arg for V any → string")
	})
}

// analyzeAndProject runs the full suite pipeline for unit testing
// [suite.Data] and [suite.MethodView] methods.
func analyzeAndProject(t *testing.T, fixture, iface string) *suite.Data {
	t.Helper()
	pkg := loadFixture(t, fixture)
	d, err := suite.Analyze(pkg, []string{iface}, generator.DefaultConfig(), generator.Options{
		Output: "test/out.gen.go",
	})
	testkit.NoError(t, err, "Analyze")
	testkit.NoError(t, suite.Enrich(d, pkg), "Enrich")
	testkit.NoError(t, suite.Project(d, pkg), "Project")
	return d
}

func TestAnalyze(t *testing.T) {
	t.Parallel()

	t.Run("error from spec.Analyze propagates", func(t *testing.T) {
		t.Parallel()
		pkg := loadFixture(t, "basic")
		_, err := suite.Analyze(pkg, []string{"DoesNotExist"},
			generator.DefaultConfig(), generator.Options{Output: "x.gen.go"})
		testkit.True(t, err != nil, "nonexistent interface rejected")
	})

	t.Run("DriverName follows Assert<Iface>Contract pattern", func(t *testing.T) {
		t.Parallel()
		d := analyzeAndProject(t, "basic", "Store")
		testkit.Equal(t, d.DriverName, "AssertStoreContract", "DriverName")
	})

	t.Run("AcrossImplsName follows pattern", func(t *testing.T) {
		t.Parallel()
		d := analyzeAndProject(t, "basic", "Store")
		testkit.Equal(t, d.AcrossImplsName, "AssertStoreContractAcrossImpls",
			"AcrossImplsName")
	})

	t.Run("FactoryTypeName follows pattern", func(t *testing.T) {
		t.Parallel()
		d := analyzeAndProject(t, "basic", "Store")
		testkit.Equal(t, d.FactoryTypeName, "StoreFactory", "FactoryTypeName")
	})

	t.Run("NamedFactoryTypeName follows pattern", func(t *testing.T) {
		t.Parallel()
		d := analyzeAndProject(t, "basic", "Store")
		testkit.Equal(t, d.NamedFactoryTypeName, "StoreNamedFactory",
			"NamedFactoryTypeName")
	})
}

func TestProject(t *testing.T) {
	t.Parallel()

	t.Run("HasErrorMethod true when any method returns error", func(t *testing.T) {
		t.Parallel()
		d := analyzeAndProject(t, "basic", "Store")
		testkit.True(t, d.HasErrorMethod, "Store.Get returns error")
	})

	t.Run("Methods populated with correct count", func(t *testing.T) {
		t.Parallel()
		d := analyzeAndProject(t, "basic", "Store")
		testkit.Len(t, d.Methods, 2, "Store has Get + Put")
	})
}

func methodByName(t *testing.T, d *suite.Data, name string) *suite.MethodView {
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

	t.Run("IfaceName returns source interface name", func(t *testing.T) {
		t.Parallel()
		d := analyzeAndProject(t, "basic", "Store")
		m := methodByName(t, d, "Get")
		testkit.Equal(t, m.IfaceName(), "Store", "IfaceName")
	})

	t.Run("ShapeName returns detected shape", func(t *testing.T) {
		t.Parallel()
		d := analyzeAndProject(t, "interfaces", "AllShapes")
		get := methodByName(t, d, "Get")
		testkit.Equal(t, get.ShapeName(), "Reader", "Get is Reader")
		put := methodByName(t, d, "Put")
		testkit.Equal(t, put.ShapeName(), "Writer", "Put is Writer")
	})

	t.Run("ShapeKeyType returns rendered key type", func(t *testing.T) {
		t.Parallel()
		d := analyzeAndProject(t, "interfaces", "AllShapes")
		get := methodByName(t, d, "Get")
		testkit.Equal(t, get.ShapeKeyType(), "string", "Get key is string")
	})

	t.Run("ShapeKeyType2 empty for single-key shapes", func(t *testing.T) {
		t.Parallel()
		d := analyzeAndProject(t, "interfaces", "AllShapes")
		get := methodByName(t, d, "Get")
		testkit.Equal(t, get.ShapeKeyType2(), "", "Reader has no KeyType2")
	})

	t.Run("ShapeValType returns rendered value type", func(t *testing.T) {
		t.Parallel()
		d := analyzeAndProject(t, "interfaces", "AllShapes")
		get := methodByName(t, d, "Get")
		testkit.True(t, get.ShapeValType() != "", "Get has a value type")
	})

	t.Run("ShapeValType2 empty for single-value shapes", func(t *testing.T) {
		t.Parallel()
		d := analyzeAndProject(t, "interfaces", "AllShapes")
		get := methodByName(t, d, "Get")
		testkit.Equal(t, get.ShapeValType2(), "", "Reader has no ValType2")
	})

	t.Run("ShapeRetType empty for non-writer-with-result", func(t *testing.T) {
		t.Parallel()
		d := analyzeAndProject(t, "interfaces", "AllShapes")
		get := methodByName(t, d, "Get")
		testkit.Equal(t, get.ShapeRetType(), "", "Reader has no RetType")
	})

	t.Run("ShapeIterValType populated for StreamReader", func(t *testing.T) {
		t.Parallel()
		d := analyzeAndProject(t, "interfaces", "AllShapes")
		all := methodByName(t, d, "All")
		testkit.True(t, all.ShapeIterValType() != "",
			"StreamReader All has an iter value type")
	})

	t.Run("ShapeIterValType empty for non-stream", func(t *testing.T) {
		t.Parallel()
		d := analyzeAndProject(t, "interfaces", "AllShapes")
		get := methodByName(t, d, "Get")
		testkit.Equal(t, get.ShapeIterValType(), "", "Reader has no iter type")
	})

	t.Run("FirstSentinel returns qualified sentinel expression", func(t *testing.T) {
		t.Parallel()
		d := analyzeAndProject(t, "interfaces", "AllShapes")
		get := methodByName(t, d, "Get")
		testkit.True(t, get.HasFirstSentinel(),
			"Get has //testkit:errors ErrNotFound")
		testkit.True(t, get.FirstSentinel() != "",
			"FirstSentinel is non-empty")
	})

	t.Run("FirstSentinel empty when no errors directive", func(t *testing.T) {
		t.Parallel()
		d := analyzeAndProject(t, "interfaces", "AllShapes")
		put := methodByName(t, d, "Put")
		testkit.False(t, put.HasFirstSentinel(), "Put has no errors directive")
		testkit.Equal(t, put.FirstSentinel(), "", "FirstSentinel is empty")
	})

	t.Run("FirstNonSkipMethod returns a non-integration-only method", func(t *testing.T) {
		t.Parallel()
		d := analyzeAndProject(t, "interfaces", "Directives")
		m := d.FirstNonSkipMethod()
		testkit.True(t, m != nil, "Directives has non-skip methods")
		testkit.False(t, m.IsIntegrationOnly(), "returned method is not skipped")
	})

	t.Run("SampleParamAt prefers //testkit:sample over default literal", func(t *testing.T) {
		t.Parallel()
		d := analyzeAndProject(t, "basic", "Sampler")
		lookup := methodByName(t, d, "Lookup")
		testkit.Equal(t, lookup.SampleParamAt(0, d.Spec.Tracker), "basic.SampleKey()",
			"SampleKey directive overrides default test-key literal")
		apply := methodByName(t, d, "Apply")
		testkit.Equal(t, apply.SampleParamAt(0, d.Spec.Tracker), "basic.SampleKey()",
			"first sample param resolves to qualified SampleKey()")
		testkit.Equal(t, apply.SampleParamAt(1, d.Spec.Tracker), "basic.SampleItem()",
			"second sample param resolves to qualified SampleItem()")
	})

	t.Run("SampleParamAt falls back to default when no //testkit:sample", func(t *testing.T) {
		t.Parallel()
		d := analyzeAndProject(t, "basic", "Store")
		get := methodByName(t, d, "Get")
		testkit.Equal(t, get.SampleParamAt(0, d.Spec.Tracker), `"test-key"`,
			"Get without sample directive uses default test-key literal")
	})

	t.Run("SampleArgs uses //testkit:sample call list when count matches", func(t *testing.T) {
		t.Parallel()
		d := analyzeAndProject(t, "basic", "Sampler")
		apply := methodByName(t, d, "Apply")
		testkit.Equal(t, apply.SampleArgs(d.Spec.Tracker),
			"t.Context(), basic.SampleKey(), basic.SampleItem()",
			"SampleArgs prepends t.Context() and joins qualified sample calls")
	})

	t.Run("SampleArgs falls back to default when no //testkit:sample", func(t *testing.T) {
		t.Parallel()
		d := analyzeAndProject(t, "basic", "Store")
		get := methodByName(t, d, "Get")
		testkit.Equal(t, get.SampleArgs(d.Spec.Tracker), `t.Context(), "test-key"`,
			"SampleArgs falls back to spec defaults")
	})
}
