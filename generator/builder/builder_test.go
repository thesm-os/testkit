// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package builder_test

import (
	"testing"

	backendgolang "go.thesmos.sh/eidos/backend/golang"
	"go.thesmos.sh/eidos/core/diag"
	"go.thesmos.sh/eidos/core/position"
	"go.thesmos.sh/eidos/eidostest/pipelinetest"
	"go.thesmos.sh/eidos/eidostest/plugintest"
	"go.thesmos.sh/eidos/eidostest/storefixture"
	"go.thesmos.sh/eidos/node"
	"go.thesmos.sh/eidos/plugin"
	"go.thesmos.sh/eidos/store"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/generator/builder"
	"go.thesmos.sh/testkit/generator/internal/defaults"
)

// The framework conformance suites pin the static contract — stable Name,
// deterministic Outputs, a well-formed multi-output shape, templates that
// parse — none of which a fixture-driven test would notice.
func TestConformance(t *testing.T) {
	t.Parallel()

	t.Run("satisfies the framework contracts", func(t *testing.T) {
		t.Parallel()
		plugintest.RunSuite(t, builder.New())
	})

	t.Run("satisfies the generator contracts", func(t *testing.T) {
		t.Parallel()
		plugintest.RunGeneratorSuite(t, builder.New(), []plugintest.GeneratorFixture{
			{
				Name:       "annotated struct",
				BuildStore: func(t *testing.T) *store.Store { t.Helper(); return fixture(t, field("Name", nil)) },
			},
			{
				Name: "struct seeded from a companion",
				BuildStore: func(t *testing.T) *store.Store {
					t.Helper()
					return withCompanion(t, "Config", "ConfigDefaults", "Config")
				},
			},
			{
				Name: "empty store",
				BuildStore: func(t *testing.T) *store.Store {
					t.Helper()
					return storefixture.New().Build()
				},
			},
		})
	})
}

// The companion lands in the external test package of wherever the builder was
// routed, so it reaches neither the builder nor the struct unqualified — and
// routing is not resolved until after Generate has run.
func TestSetOutputPackages(t *testing.T) {
	t.Parallel()

	t.Run("repoints the constructors at where the builder was routed", func(t *testing.T) {
		t.Parallel()
		tests := &builder.Tests{SourceName: "Config"}
		tests.SetOutputPackages(map[string]string{"": "example.com/cfg/cfgtest"})
		testkit.Equal(t, tests.CtorRef.Pkg, "example.com/cfg/cfgtest", "the reference follows the routing")
		testkit.Equal(t, tests.FromRef.Name, "NewConfigFrom", "the seeding constructor routes with it")
	})

	t.Run("tolerates routing that resolved no path", func(t *testing.T) {
		t.Parallel()
		// Layout reaches dispatch with some tags missing when a run recorded
		// routing errors, so the map is not always complete.
		tests := &builder.Tests{SourceName: "Config"}
		tests.SetOutputPackages(map[string]string{})
		tests.SetOutputPackages(map[string]string{"": ""})
		testkit.True(t, tests.CtorRef == nil, "an underivable path leaves the reference alone")
	})
}

// A struct's embedded types are recorded apart from its declared fields, so a
// projection reading only the latter offers no way to set them at all.
func TestEmbedded(t *testing.T) {
	t.Parallel()

	t.Run("sets an embedded type as a whole", func(t *testing.T) {
		t.Parallel()
		// Promoting the fields inside it would offer two ways to write the same
		// thing that disagree about whether the embedded value is set.
		f := render(t, embeddedPackage()).AssertFile("types" + builder.GoPrimarySuffix)
		f.Contains("func (b *ItemBuilder) WithMeta(v Meta) *ItemBuilder")
	})
}

// The explicit companion key exists for one that does not follow the
// convention or does not live beside the struct.
func TestExplicitCompanion(t *testing.T) {
	t.Parallel()

	t.Run("calls a companion named by full import path", func(t *testing.T) {
		t.Parallel()
		// A companion elsewhere would otherwise need an import written only for
		// this directive, which does not compile.
		render(t, companionPackage()).AssertFile("types" + builder.GoPrimarySuffix).
			Contains("seed.Seed()")
	})
}

// The substitution is what lets a generic builder's checks be an ordinary
// non-generic test function, and it has to reach inside composites.
func TestGenericSubstitution(t *testing.T) {
	t.Parallel()

	t.Run("rewrites a type parameter inside a slice", func(t *testing.T) {
		t.Parallel()
		render(t, genericCompositePackage()).AssertFile("types" + builder.GoTestSuffix).
			Contains("var v string")
	})

	t.Run("rewrites a type parameter inside a map", func(t *testing.T) {
		t.Parallel()
		render(t, genericCompositePackage()).AssertFile("types" + builder.GoTestSuffix).
			Contains("map[string]string")
	})
}

// embeddedPackage carries a struct embedding another type.
func embeddedPackage() *node.Package {
	return storefixture.New().
		Package("cfg", "example.com/cfg").
		Struct("Item", func(b *storefixture.StructBuilder) {
			b.Pos(position.At("cfg/types.go", 1, 1))
			b.Directive(storefixture.Directive("builder"))
			b.Embed(storefixture.Named("Meta"))
			b.Field("Name", storefixture.Named("string"), nil)
		}).
		PackageNode()
}

// companionPackage names its companion by full import path.
func companionPackage() *node.Package {
	return storefixture.New().
		Package("cfg", "example.com/cfg").
		Struct("Item", func(b *storefixture.StructBuilder) {
			b.Pos(position.At("cfg/types.go", 1, 1))
			b.Directive(storefixture.Directive("builder",
				storefixture.KV("defaults", "example.com/seed.Seed")))
			b.Field("Name", storefixture.Named("string"), nil)
		}).
		PackageNode()
}

// genericCompositePackage parameterises a slice and a map.
func genericCompositePackage() *node.Package {
	return storefixture.New().
		Package("cfg", "example.com/cfg").
		Struct("Box", func(b *storefixture.StructBuilder) {
			b.Pos(position.At("cfg/types.go", 1, 1))
			b.Directive(storefixture.Directive("builder"))
			b.TypeParam("T", bound("any"))
			b.Field("Items", storefixture.Slice(storefixture.Named("T")), nil)
			b.Field("By", storefixture.Map(storefixture.Named("T"), storefixture.Named("T")), nil)
		}).
		PackageNode()
}

// The diagnostics are the one behaviour the corpus cannot show: a fixture that
// provokes one would fail the run that generates every other fixture.
func TestDiagnostics(t *testing.T) {
	t.Parallel()

	t.Run("reports a struct with nothing to set", func(t *testing.T) {
		t.Parallel()
		// A builder with no setters configures nothing, and emitting the shell
		// would hide a declaration that cannot do what it says.
		diags := diagnostics(t, fixture(t, field("secret", nil)))
		testkit.Len(t, diags, 1, "a builder with no fields is reported")
	})

	t.Run("rejects a tag value that is not the opt-out", func(t *testing.T) {
		t.Parallel()
		// Silently keeping the setter would leave the author believing a field
		// they meant to exclude is excluded.
		diags := diagnostics(t, fixture(t, tagged("Name", `builder:"skip"`)))
		testkit.Len(t, diags, 1, "a mistyped opt-out is reported")
	})
}

// field returns a builder option declaring one string field.
func field(name string, _ any) func(*storefixture.StructBuilder) {
	return func(b *storefixture.StructBuilder) {
		b.Field(name, storefixture.Named("string"), nil)
	}
}

// tagged returns a builder option declaring one string field carrying tag.
func tagged(name, tag string) func(*storefixture.StructBuilder) {
	return func(b *storefixture.StructBuilder) {
		b.Field(name, storefixture.Named("string"), func(f *storefixture.FieldBuilder) {
			f.Tag(tag)
		})
	}
}

// fixture returns a store holding one annotated struct assembled from opts.
func fixture(t *testing.T, opts ...func(*storefixture.StructBuilder)) *store.Store {
	t.Helper()
	return storefixture.New().
		Package("cfg", "example.com/cfg").
		Struct("Config", func(b *storefixture.StructBuilder) {
			b.Directive(storefixture.Directive("builder"))
			for _, opt := range opts {
				opt(b)
			}
		}).
		Build()
}

// withCompanion returns a store holding an annotated struct and a function
// named companion returning returns.
func withCompanion(t *testing.T, name, companion, returns string) *store.Store {
	t.Helper()
	return storefixture.New().
		Package("cfg", "example.com/cfg").
		Struct(name, func(b *storefixture.StructBuilder) {
			b.Directive(storefixture.Directive("builder"))
			b.Field("Host", storefixture.Named("string"), nil)
		}).
		Function(companion, func(f *storefixture.FunctionBuilder) {
			f.Return(storefixture.Named(returns))
		}).
		Build()
}

// diagnostics drives the plugin over s and returns what it reported.
func diagnostics(t *testing.T, s *store.Store) []diag.Diag {
	t.Helper()
	sink := diag.New()
	if err := builder.New().Generate(context(s, sink)); err != nil {
		t.Fatalf("Generate: %v", err)
	}
	return sink.Diagnostics()
}

// context assembles the generator context the plugin reads through.
func context(s *store.Store, d *diag.Sink) *plugin.GeneratorContext {
	return &plugin.GeneratorContext{Store: s, Reader: store.NewReader(s), Diag: d}
}

// Rendering is where a generator actually fails. Every assertion driven off the
// emit graph passes against a template that renders code which does not compile
// — an undeclared local, a setter whose receiver disagrees with its return.
// Those surface only once the backend runs, so the templates are driven
// end-to-end here rather than trusted.
func TestRender(t *testing.T) {
	t.Parallel()

	t.Run("renders both outputs", func(t *testing.T) {
		t.Parallel()
		render(t, plainPackage()).AssertFileCount(2)
	})

	t.Run("renders without a diagnostic", func(t *testing.T) {
		t.Parallel()
		testkit.Len(t, render(t, plainPackage()).Diagnostics().Diagnostics(), 0,
			"a clean fixture renders without diagnostics")
	})

	t.Run("emits a setter shaped by each field's type", func(t *testing.T) {
		t.Parallel()
		f := render(t, plainPackage()).AssertFile("types" + builder.GoPrimarySuffix)
		for _, want := range []string{
			"func (b *ItemBuilder) WithName(v string) *ItemBuilder",
			"func (b *ItemBuilder) WithTags(v ...string) *ItemBuilder",
			"func (b *ItemBuilder) AppendTags(v ...string) *ItemBuilder",
			"func (b *ItemBuilder) WithBodyString(s string) *ItemBuilder",
			"func (b *ItemBuilder) WithMetaEntry(k string, v string) *ItemBuilder",
			"func (b *ItemBuilder) WithMetaEntries(entries map[string]string) *ItemBuilder",
			"func (b *ItemBuilder) Clone() *ItemBuilder",
		} {
			f.Contains(want)
		}
	})

	t.Run("copies every field that owns storage", func(t *testing.T) {
		t.Parallel()
		// A clone sharing a slice or map lets one test's setup appear in
		// another's, which surfaces as a failure for something it never did.
		f := render(t, plainPackage()).AssertFile("types" + builder.GoPrimarySuffix)
		f.Contains("out.v.Tags = append([]string(nil), b.v.Tags...)")
		f.Contains("out.v.Meta = make(map[string]string, len(b.v.Meta))")
	})

	t.Run("seeds the constructor from a declared default", func(t *testing.T) {
		t.Parallel()
		render(t, seededPackage()).AssertFile("types" + builder.GoPrimarySuffix).
			Contains(`Name: "seed"`)
	})

	t.Run("instantiates a generic builder's checks at concrete types", func(t *testing.T) {
		t.Parallel()
		// A Go test function cannot take type parameters, so a check naming the
		// parameter in a field position would not compile.
		render(t, genericPackage()).AssertFile("types" + builder.GoTestSuffix).
			Contains("NewBox[string]()")
	})

	t.Run("declines to check a builder it cannot instantiate", func(t *testing.T) {
		t.Parallel()
		render(t, boundedPackage()).AssertFile("types" + builder.GoTestSuffix).
			Contains("Skipped:")
	})
}

// render drives the plugin and the Go backend over pkg through a synthetic
// pipeline, so routing and rendering both participate.
func render(t *testing.T, pkg *node.Package) *pipelinetest.Pipeline {
	t.Helper()
	return pipelinetest.New(t).
		WithFrontend(pipelinetest.FromNodes(pkg)).
		WithGenerator(builder.New()).
		WithBackend(backendgolang.New()).
		Build().
		Run()
}

// plainPackage carries one field of every shape that changes a setter.
func plainPackage() *node.Package {
	return storefixture.New().
		Package("cfg", "example.com/cfg").
		Struct("Item", func(b *storefixture.StructBuilder) {
			// Layout composes the filename from the source basename, so the
			// fixture needs a position for the rendered name to be anything
			// other than a bare suffix.
			b.Pos(position.At("cfg/types.go", 1, 1))
			b.Directive(storefixture.Directive("builder"))
			b.Field("Name", storefixture.Named("string"), nil)
			b.Field("Tags", storefixture.Slice(storefixture.Named("string")), nil)
			b.Field("Body", storefixture.Slice(storefixture.Named("byte")), nil)
			b.Field("Meta", storefixture.Map(storefixture.Named("string"), storefixture.Named("string")), nil)
			b.Field("hidden", storefixture.Named("string"), nil)
		}).
		PackageNode()
}

// seededPackage carries a field declaring a default.
func seededPackage() *node.Package {
	pkg := storefixture.New().
		Package("cfg", "example.com/cfg").
		Struct("Item", func(b *storefixture.StructBuilder) {
			b.Pos(position.At("cfg/types.go", 1, 1))
			b.Directive(storefixture.Directive("builder"))
			b.Field("Name", storefixture.Named("string"), nil)
		}).
		PackageNode()
	for _, s := range pkg.Structs {
		for _, f := range s.Fields {
			defaults.MetaDefault.Set(f.EnsureMeta(), `"seed"`, "test")
		}
	}
	return pkg
}

// genericPackage carries a struct whose constraints admit a witness.
func genericPackage() *node.Package {
	return storefixture.New().
		Package("cfg", "example.com/cfg").
		Struct("Box", func(b *storefixture.StructBuilder) {
			b.Pos(position.At("cfg/types.go", 1, 1))
			b.Directive(storefixture.Directive("builder"))
			b.TypeParam("T", bound("any"))
			b.Field("Value", storefixture.Named("T"), nil)
		}).
		PackageNode()
}

// boundedPackage carries a struct bounded by a constraint no witness satisfies.
func boundedPackage() *node.Package {
	return storefixture.New().
		Package("cfg", "example.com/cfg").
		Struct("Ranked", func(b *storefixture.StructBuilder) {
			b.Pos(position.At("cfg/types.go", 1, 1))
			b.Directive(storefixture.Directive("builder"))
			b.TypeParam("K", bound("Ordered"))
			b.Field("Key", storefixture.Named("K"), nil)
		}).
		PackageNode()
}

// bound builds the constraint the frontend produces for a written bound.
func bound(name string) *node.Constraint {
	c := storefixture.Constraint(storefixture.Named(name))
	c.Raw = name
	return c
}
