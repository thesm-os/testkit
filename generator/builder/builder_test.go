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

	t.Run("sets a type embedded by pointer", func(t *testing.T) {
		t.Parallel()
		// An embed by pointer records its name on the pointee, so a projection
		// reading the reference's own name drops the field with no diagnostic
		// and the promoted fields become unreachable.
		f := render(t, embeddedPackage()).AssertFile("types" + builder.GoPrimarySuffix)
		f.Contains("func (b *ItemBuilder) WithAudit(v Audit) *ItemBuilder")
	})

	t.Run("allocates for a type embedded by pointer", func(t *testing.T) {
		t.Parallel()
		// The promoted fields are reachable only once the pointer is non-nil, so
		// a setter demanding an address makes every caller allocate first.
		f := render(t, embeddedPackage()).AssertFile("types" + builder.GoPrimarySuffix)
		f.Contains("b.v.Audit = &v")
	})
}

// A check comparing a field against the zero value passes against a setter that
// assigns nothing, which is what the sample pair exists to prevent — so what is
// under test is that the pair reaches the rendered check, and that a type with
// no honest pair loses the check rather than keeping a vacuous one.
func TestSamples(t *testing.T) {
	t.Parallel()

	t.Run("sets a field to a value distinct from its zero", func(t *testing.T) {
		t.Parallel()
		render(t, plainPackage()).AssertFile("types" + builder.GoTestSuffix).
			Contains(`WithName("test-name").Build().Name, "test-name"`)
	})

	t.Run("sets it a second time to a different value", func(t *testing.T) {
		t.Parallel()
		// One value passes whenever the constructor already seeded it, and a
		// companion's return is opaque here — the pair is what covers that.
		render(t, plainPackage()).AssertFile("types" + builder.GoTestSuffix).
			Contains(`WithName("other-name").Build().Name, "other-name"`)
	})

	t.Run("omits the check for a type admitting no pair", func(t *testing.T) {
		t.Parallel()
		// Keeping it would assert, pass, and prove nothing, which reads as
		// coverage the setter does not have.
		render(t, embeddedPackage()).AssertFile("types" + builder.GoTestSuffix).
			NotContains(`t.Run("reaches Meta"`)
	})

	t.Run("says why the check is absent", func(t *testing.T) {
		t.Parallel()
		// A reader looking for the check finds the reason rather than a gap.
		render(t, embeddedPackage()).AssertFile("types" + builder.GoTestSuffix).
			Contains("No check that the setter reaches Meta")
	})

	t.Run("derives a pair once a type parameter resolves to its witness", func(t *testing.T) {
		t.Parallel()
		// The parameter admits no pair at the source, so a projection that did
		// not re-derive after substitution would leave every generic field's
		// setter unchecked.
		render(t, genericPackage()).AssertFile("types" + builder.GoTestSuffix).
			Contains(`WithValue("test-value")`)
	})
}

// Three field types carry no value that can be written down, and each is
// checkable anyway by a route of its own — which is what keeps them from
// falling into the bucket of setters nothing asserts about.
func TestUnwritableFields(t *testing.T) {
	t.Parallel()

	t.Run("checks a channel by identity", func(t *testing.T) {
		t.Parallel()
		// A freshly made channel is distinct from anything the constructor
		// could have seeded, so one value proves what a comparable type needs
		// two for.
		f := render(t, unwritablePackage()).AssertFile("types" + builder.GoTestSuffix)
		f.Contains("ch := make(chan string)")
		f.Contains("WithEvents(ch).Build().Events == ch")
	})

	t.Run("checks a func by arrival", func(t *testing.T) {
		t.Parallel()
		// A func is not comparable, so there is nothing else to assert — but a
		// setter assigning nothing leaves nil, which this catches.
		f := render(t, unwritablePackage()).AssertFile("types" + builder.GoTestSuffix)
		f.Contains("Build().Callback != nil")
	})

	t.Run("gives the func literal a body returning its own zero values", func(t *testing.T) {
		t.Parallel()
		// A literal is the only non-nil func available, and its body has to
		// satisfy whatever the field's signature returns.
		f := render(t, unwritablePackage()).AssertFile("types" + builder.GoTestSuffix)
		f.Contains("var r0 error")
		f.Contains("return r0")
	})

	t.Run("checks an error by identity", func(t *testing.T) {
		t.Parallel()
		// Two errors carrying the same text are not equal, so the check matches
		// the one it handed over rather than comparing values.
		f := render(t, unwritablePackage()).AssertFile("types" + builder.GoTestSuffix)
		f.Contains(`errors.New("test-Err")`)
		f.Contains("testkit.ErrorIs(")
	})

	t.Run("declines a type from a package the run never read", func(t *testing.T) {
		t.Parallel()
		// The floor: nothing about time.Time is in the graph, so no value of it
		// can be written and the check is dropped rather than faked.
		render(t, unwritablePackage()).AssertFile("types" + builder.GoTestSuffix).
			Contains("No check that the setter reaches At")
	})

	t.Run("declines a directional channel", func(t *testing.T) {
		t.Parallel()
		// make is not legal on a receive-only channel, so a check that treated
		// every channel alike would emit code that does not compile — and the
		// direction is in the stamp, not in the reference's shape.
		render(t, directionalChanPackage()).AssertFile("types" + builder.GoTestSuffix).
			NotContains("make(")
	})
}

// directionalChanPackage carries a receive-only channel, which takes a setter
// like any other but no check that has to construct one.
func directionalChanPackage() *node.Package {
	return storefixture.New().
		Package("cfg", "example.com/cfg").
		Struct("Item", func(b *storefixture.StructBuilder) {
			b.Pos(position.At("cfg/types.go", 1, 1))
			b.Directive(storefixture.Directive("builder"))
			b.Field("Events", chanRef("recv"), nil)
		}).
		PackageNode()
}

// chanRef builds a channel the way the Go frontend does: a named reference in a
// synthetic `go` package carrying the element as its type argument, with the
// facts that make it a channel stamped beside it rather than in its shape.
func chanRef(dir string) *node.TypeRef {
	ref := storefixture.WithArgs(storefixture.PkgNamed("go", "chan"), storefixture.Named("string"))
	builder.GoIsChannel.Set(ref.EnsureMeta(), true, "golang")
	builder.GoChanDir.Set(ref.EnsureMeta(), dir, "golang")
	return ref
}

// unwritablePackage carries the field types no literal can be written for.
func unwritablePackage() *node.Package {
	events := chanRef(builder.ChanBidirectional)
	return storefixture.New().
		Package("cfg", "example.com/cfg").
		Struct("Item", func(b *storefixture.StructBuilder) {
			b.Pos(position.At("cfg/types.go", 1, 1))
			b.Directive(storefixture.Directive("builder"))
			b.Field("Events", events, nil)
			b.Field("Callback", storefixture.Func(
				[]*node.TypeRef{storefixture.Named("int")},
				[]*node.TypeRef{storefixture.Named("error")},
			), nil)
			b.Field("Err", storefixture.Named("error"), nil)
			b.Field("At", storefixture.PkgNamed("time", "Time"), nil)
		}).
		PackageNode()
}

// A map to the empty struct carries its whole meaning in its keys, so a setter
// asking for the value asks the caller for the one thing they cannot vary.
func TestSetField(t *testing.T) {
	t.Parallel()

	t.Run("takes no value parameter on the entry setter", func(t *testing.T) {
		t.Parallel()
		render(t, setPackage()).AssertFile("types" + builder.GoPrimarySuffix).
			Contains("func (b *ItemBuilder) WithTagsEntry(k string) *ItemBuilder")
	})

	t.Run("adds many keys variadically", func(t *testing.T) {
		t.Parallel()
		// A caller writing map[string]struct{}{"a": {}, "b": {}} at every call
		// site is the reason this shape exists at all.
		render(t, setPackage()).AssertFile("types" + builder.GoPrimarySuffix).
			Contains("func (b *ItemBuilder) WithTagsEntries(keys ...string) *ItemBuilder")
	})

	t.Run("supplies the value itself", func(t *testing.T) {
		t.Parallel()
		render(t, setPackage()).AssertFile("types" + builder.GoPrimarySuffix).
			Contains("b.v.Tags[k] = struct{}{}")
	})

	t.Run("copies the set on clone", func(t *testing.T) {
		t.Parallel()
		// A set is a map, so a clone sharing it lets one test's keys appear in
		// another's.
		render(t, setPackage()).AssertFile("types" + builder.GoPrimarySuffix).
			Contains("out.v.Tags = make(map[string]struct{}, len(b.v.Tags))")
	})

	t.Run("leaves a map with a real value type alone", func(t *testing.T) {
		t.Parallel()
		// The narrower reading applies only where the value is the anonymous
		// empty struct; anything else is an ordinary mapping.
		render(t, setPackage()).AssertFile("types" + builder.GoPrimarySuffix).
			Contains("func (b *ItemBuilder) WithMetaEntry(k string, v string) *ItemBuilder")
	})

	t.Run("checks the set with two distinct keys", func(t *testing.T) {
		t.Parallel()
		render(t, setPackage()).AssertFile("types" + builder.GoTestSuffix).
			Contains(`WithTagsEntry("test-tags").`)
	})
}

// A set whose key admits no sample pair is the one case the corpus cannot show,
// since its own set is keyed by string.
func TestSetFieldWithoutASamplePair(t *testing.T) {
	t.Parallel()

	t.Run("falls back to a declared zero key", func(t *testing.T) {
		t.Parallel()
		// The checks still have to compile, and a key type this generator
		// cannot write a literal for still has a zero value.
		render(t, opaqueSetPackage()).AssertFile("types" + builder.GoTestSuffix).
			Contains("var k Kind")
	})

	t.Run("omits the check two keys would be needed for", func(t *testing.T) {
		t.Parallel()
		// Adding one key twice cannot tell an adding setter from a replacing
		// one, so the check would pass against either.
		render(t, opaqueSetPackage()).AssertFile("types" + builder.GoTestSuffix).
			NotContains(`t.Run("keeps keys it was not given"`)
	})

	t.Run("says why the check is absent", func(t *testing.T) {
		t.Parallel()
		render(t, opaqueSetPackage()).AssertFile("types" + builder.GoTestSuffix).
			Contains("No check that adding keeps what was there")
	})
}

// setPackage carries a set beside an ordinary map, which take different
// setters and are told apart by the map's value type rather than by its name.
func setPackage() *node.Package {
	return storefixture.New().
		Package("cfg", "example.com/cfg").
		Struct("Item", func(b *storefixture.StructBuilder) {
			b.Pos(position.At("cfg/types.go", 1, 1))
			b.Directive(storefixture.Directive("builder"))
			b.Field("Tags", storefixture.Map(storefixture.Named("string"), emptyStruct()), nil)
			b.Field("Meta", storefixture.Map(storefixture.Named("string"), storefixture.Named("string")), nil)
		}).
		PackageNode()
}

// opaqueSetPackage keys its set by a type no sample pair can be written for.
func opaqueSetPackage() *node.Package {
	return storefixture.New().
		Package("cfg", "example.com/cfg").
		Struct("Item", func(b *storefixture.StructBuilder) {
			b.Pos(position.At("cfg/types.go", 1, 1))
			b.Directive(storefixture.Directive("builder"))
			b.Field("Kinds", storefixture.Map(storefixture.Named("Kind"), emptyStruct()), nil)
		}).
		PackageNode()
}

// emptyStruct builds the anonymous `struct{}` the frontend records for a set's
// value type.
func emptyStruct() *node.TypeRef { return storefixture.AnonStruct(nil, nil) }

// A pointer field distinguishes unset from zero, and the caller who wants to
// say "set" should not have to produce an address to say it.
func TestPointerField(t *testing.T) {
	t.Parallel()

	t.Run("takes the pointee by value", func(t *testing.T) {
		t.Parallel()
		render(t, pointerPackage()).AssertFile("types" + builder.GoPrimarySuffix).
			Contains("func (b *ItemBuilder) WithRetries(v int) *ItemBuilder")
	})

	t.Run("takes the address itself", func(t *testing.T) {
		t.Parallel()
		render(t, pointerPackage()).AssertFile("types" + builder.GoPrimarySuffix).
			Contains("b.v.Retries = &v")
	})

	t.Run("checks the field through the pointer rather than past it", func(t *testing.T) {
		t.Parallel()
		// A setter that assigned nothing leaves nil, and dereferencing that
		// panics instead of saying which setter failed.
		render(t, pointerPackage()).AssertFile("types" + builder.GoTestSuffix).
			Contains("Build().Retries, &want")
	})

	t.Run("checks that the setter allocated at all", func(t *testing.T) {
		t.Parallel()
		// The one assertion that holds for a pointee admitting no sample pair,
		// which is every pointer to a struct or an interface.
		render(t, pointerPackage()).AssertFile("types" + builder.GoTestSuffix).
			Contains("Build().Retries != nil")
	})

	t.Run("leaves a pointer element inside a slice alone", func(t *testing.T) {
		t.Parallel()
		// The rule applies to a field whose own type is a pointer. An element
		// type is the caller's to supply, so Append keeps taking it as declared.
		render(t, pointerPackage()).AssertFile("types" + builder.GoPrimarySuffix).
			Contains("func (b *ItemBuilder) AppendPeers(v ...*Item) *ItemBuilder")
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

// embeddedPackage carries a struct embedding another type both ways.
func embeddedPackage() *node.Package {
	return storefixture.New().
		Package("cfg", "example.com/cfg").
		Struct("Item", func(b *storefixture.StructBuilder) {
			b.Pos(position.At("cfg/types.go", 1, 1))
			b.Directive(storefixture.Directive("builder"))
			b.Embed(storefixture.Named("Meta"))
			b.Embed(storefixture.Pointer(storefixture.Named("Audit")))
			b.Field("Name", storefixture.Named("string"), nil)
		}).
		PackageNode()
}

// pointerPackage carries a pointer field and a slice of pointers, which take
// different setters: the rule applies to a field that is itself a pointer.
func pointerPackage() *node.Package {
	return storefixture.New().
		Package("cfg", "example.com/cfg").
		Struct("Item", func(b *storefixture.StructBuilder) {
			b.Pos(position.At("cfg/types.go", 1, 1))
			b.Directive(storefixture.Directive("builder"))
			b.Field("Retries", storefixture.Pointer(storefixture.Named("int")), nil)
			b.Field("Peers", storefixture.Slice(storefixture.Pointer(storefixture.Named("Item"))), nil)
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
