// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package stub_test

import (
	"maps"
	"path/filepath"
	"strings"
	"testing"

	backendgolang "go.thesmos.sh/eidos/backend/golang"
	"go.thesmos.sh/eidos/eidostest/golangtest"
	"go.thesmos.sh/eidos/eidostest/pipelinetest"
	"go.thesmos.sh/eidos/eidostest/plugintest"
	"go.thesmos.sh/eidos/eidostest/storefixture"
	"go.thesmos.sh/eidos/lang/golang"
	"go.thesmos.sh/eidos/plugins/annotator/shape"
	"go.thesmos.sh/eidos/sdk"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/generator/internal/gentest"
	"go.thesmos.sh/testkit/generator/stub"
)

// The framework conformance suites. The framework checks pin the static
// contract — stable Name, role implementation, deterministic Outputs,
// well-formed multi-output shape — and the generator suite pins determinism,
// a frozen source store, and diagnostic discipline across a fixture set.
func TestConformance(t *testing.T) {
	t.Parallel()

	t.Run("satisfies the framework contracts", func(t *testing.T) {
		t.Parallel()
		plugintest.RunSuite(t, stub.New())
	})

	t.Run("satisfies the generator contracts", func(t *testing.T) {
		t.Parallel()
		plugintest.RunGeneratorSuite(t, stub.New(), []plugintest.GeneratorFixture{
			{
				Name:       "annotated interface",
				BuildStore: func(t *testing.T) *sdk.Store { t.Helper(); return storeFixture(t) },
			},
			{
				Name: "empty store",
				BuildStore: func(t *testing.T) *sdk.Store {
					t.Helper()
					return storefixture.New().Build()
				},
			},
		})
	})

	t.Run("round-trips its options", func(t *testing.T) {
		t.Parallel()
		plugintest.RunOptionsSuite(t, stub.New(), plugintest.OptionsFixture{
			Valid:      map[string]string{"suffix": "Double"},
			UnknownKey: "no_such_field",
		})
	})
}

// The version composes into the plugin's cache key, so a stale one serves
// output produced by a plugin that has since changed. It also renders into
// every generated file's header, which is why it is a deliberate constant
// rather than derived from content.
func TestVersion(t *testing.T) {
	t.Parallel()

	t.Run("declares a non-empty version", func(t *testing.T) {
		t.Parallel()
		// The empty string is legal and means "never invalidate anything",
		// which is a staleness bug waiting for the first behavioural change.
		testkit.Assert(t, stub.New().Version()).
			IsNotEmpty("a plugin without a version cannot invalidate its cache")
	})

	t.Run("reports the declared constant", func(t *testing.T) {
		t.Parallel()
		testkit.Equal(t, stub.New().Version(), stub.Version, "the method reports the constant")
	})
}

// Outputs is a contract with the Layout phase: the tag routes an emit value
// to its file, and the suffixes are what Layout appends to the source
// basename. A silent change writes generated code to a path nothing reads.
func TestOutputs(t *testing.T) {
	t.Parallel()

	t.Run("declares a primary and a tagged companion for go", func(t *testing.T) {
		t.Parallel()
		got := stub.New().Outputs("golang")
		testkit.Len(t, got, 2, "golang must declare both outputs")
	})

	t.Run("puts the untagged primary first", func(t *testing.T) {
		t.Parallel()
		// The framework reserves the empty tag for a plugin's primary output
		// and requires it at index 0 when present.
		testkit.Equal(t, stub.New().Outputs("golang")[0].Tag, "", "the primary carries the empty tag")
	})

	t.Run("tags the companion so it routes independently", func(t *testing.T) {
		t.Parallel()
		testkit.Equal(t, stub.New().Outputs("golang")[1].Tag, stub.GoTestOutputTag, "companion tag")
	})

	t.Run("dispatches to the go adapter", func(t *testing.T) {
		t.Parallel()
		testkit.Equal(t, stub.New().Outputs("golang")[0].Suffix, stub.GoPrimarySuffix, "adapter suffix")
	})

	t.Run("declares no routable output for an unknown language", func(t *testing.T) {
		t.Parallel()
		testkit.Assert(t, stub.New().Outputs("rust")).IsNil("a non-Go backend gets no Go-shaped filenames")
	})
}

// A template the backend cannot resolve renders nothing and fails nowhere, so
// the language dispatch is pinned rather than assumed.
func TestTemplates(t *testing.T) {
	t.Parallel()

	t.Run("ships a tree for go", func(t *testing.T) {
		t.Parallel()
		_, ok := stub.New().Templates("golang")
		testkit.True(t, ok, "golang must ship a template tree")
	})

	t.Run("ships none for an unknown language", func(t *testing.T) {
		t.Parallel()
		_, ok := stub.New().Templates("rust")
		testkit.False(t, ok, "a non-Go backend gets no templates")
	})
}

// The funcmap is what the templates call; an empty one surfaces as a template
// execution error rather than as missing output.
func TestTemplateFuncs(t *testing.T) {
	t.Parallel()

	t.Run("carries the shared go helpers", func(t *testing.T) {
		t.Parallel()
		testkit.Assert(t, stub.New().TemplateFuncs("golang")).IsNotEmpty("golang needs the shared helpers")
	})

	t.Run("returns nothing for an unknown language", func(t *testing.T) {
		t.Parallel()
		testkit.Assert(t, stub.New().TemplateFuncs("rust")).IsNil("a non-Go backend gets no helpers")
	})
}

// Overriding a canonical entry changes rendering for every plugin sharing the
// backend, so the empty return is a deliberate contract rather than an
// unimplemented stub.
func TestTemplateOverrides(t *testing.T) {
	t.Parallel()

	testkit.Assert(t, stub.New().TemplateOverrides("golang")).IsNil("the plugin replaces no canonical entry")
}

func TestGenerate(t *testing.T) {
	t.Parallel()

	t.Run("queues one double and one companion per annotated interface", func(t *testing.T) {
		t.Parallel()
		testkit.Len(t, generate(t, stub.New(), storeFixture(t)), 2, "one contribution per output")
	})

	t.Run("reports the double under the primary kind", func(t *testing.T) {
		t.Parallel()
		double, _ := split(t, generate(t, stub.New(), storeFixture(t)))
		testkit.Equal(t, double.Kind(), stub.KindStub, "primary emit kind")
	})

	t.Run("reports the companion under the test kind", func(t *testing.T) {
		t.Parallel()
		_, tests := split(t, generate(t, stub.New(), storeFixture(t)))
		testkit.Equal(t, tests.Kind(), stub.KindStubTests, "companion emit kind")
	})

	t.Run("leaves the double untagged so it lands in the primary file", func(t *testing.T) {
		t.Parallel()
		double, _ := split(t, generate(t, stub.New(), storeFixture(t)))
		testkit.Equal(t, double.OutputTag(), "", "the primary output carries no tag")
	})

	t.Run("tags the companion so it routes independently", func(t *testing.T) {
		t.Parallel()
		_, tests := split(t, generate(t, stub.New(), storeFixture(t)))
		testkit.Equal(t, tests.OutputTag(), stub.GoTestOutputTag, "companion output tag")
	})

	t.Run("appends both contributions to the same file slot", func(t *testing.T) {
		t.Parallel()
		for _, p := range generate(t, stub.New(), storeFixture(t)) {
			testkit.Equal(t, p.SlotName, stub.SlotName, "every contribution shares the slot")
		}
	})

	t.Run("skips an interface without the directive", func(t *testing.T) {
		t.Parallel()
		s := storefixture.New().
			Interface("Plain", func(i *storefixture.InterfaceBuilder) {
				i.Method("Do", nil)
			}).
			Build()
		testkit.Assert(t, generate(t, stub.New(), s)).IsEmpty("an unannotated interface generates nothing")
	})

	t.Run("reports an annotated interface with no methods", func(t *testing.T) {
		t.Parallel()
		// A double with nothing to stand in for is a mistake. Emitting an
		// empty struct would compile and hide it, so the plugin reports it.
		s := emptyInterfaceFixture()
		d := sdk.NewSink()

		_ = stub.New().Generate(generatorContext(s, d))

		testkit.True(t, d.HasErrors(), "an interface with no methods must be diagnosed")
	})

	t.Run("continues the run after reporting an empty interface", func(t *testing.T) {
		t.Parallel()
		// A diagnosed fixture is skipped, not fatal: one bad interface must
		// not stop every other interface in the run from generating.
		err := stub.New().Generate(generatorContext(emptyInterfaceFixture(), sdk.NewSink()))

		testkit.NoError(t, err, "a reported interface must not abort the run")
	})

	t.Run("queues nothing for an annotated interface with no methods", func(t *testing.T) {
		t.Parallel()
		s := emptyInterfaceFixture()

		_ = stub.New().Generate(generatorContext(s, sdk.NewSink()))

		testkit.Assert(t, s.Emit().PendingOriginSlots()).IsEmpty("a diagnosed interface emits nothing")
	})

	t.Run("carries an integration-only method like any other", func(t *testing.T) {
		t.Parallel()
		// The mixin is a law for the suite and model tiers about what a test
		// runs against. It says nothing about the double, and a double missing
		// the method would not satisfy the interface at all — which is the one
		// thing a double has to do.
		double, _ := split(t, generate(t, stub.New(), mixedFixture(t)))
		testkit.Len(t, double.Methods, 2, "every declared method is doubled")
	})

	t.Run("reports an interface with no methods at all", func(t *testing.T) {
		t.Parallel()
		// Measured after projection: the interface has a method, but nothing
		// a double can stand in for, and an empty shell would satisfy and
		// record nothing.
		s := emptyFixture()
		d := sdk.NewSink()

		_ = stub.New().Generate(generatorContext(s, d))

		testkit.True(t, d.HasErrors(), "an all-integration interface must be diagnosed")
	})

	t.Run("carries the order constraint from the mixin parameter", func(t *testing.T) {
		t.Parallel()
		double, _ := split(t, generate(t, stub.New(), orderedFixture()))
		for _, m := range double.Methods {
			if m.Name == "Commit" {
				testkit.Equal(t, m.OrderAfter, "Prepare", "the prerequisite comes from the mixin")
				return
			}
		}
		t.Fatal("fixture has no Commit method")
	})

	t.Run("reports the double as ordered when any method is constrained", func(t *testing.T) {
		t.Parallel()
		// The tracker is allocated only when something needs it, so this is
		// what decides whether the double carries one at all.
		double, _ := split(t, generate(t, stub.New(), orderedFixture()))
		testkit.True(t, double.Ordered(), "a constrained method makes the double ordered")
	})

	t.Run("reports the double as unordered when nothing is constrained", func(t *testing.T) {
		t.Parallel()
		double, _ := split(t, generate(t, stub.New(), storeFixture(t)))
		testkit.False(t, double.Ordered(), "an unconstrained double needs no tracker")
	})

	t.Run("names the double with the configured suffix", func(t *testing.T) {
		t.Parallel()
		double, _ := split(t, generate(t, withSuffix(t, "Double"), storeFixture(t)))
		testkit.Equal(t, double.TypeName, "StoreDouble", "the suffix option names the type")
	})

	t.Run("falls back to the default suffix when one is set empty", func(t *testing.T) {
		t.Parallel()
		// Taken literally, an empty suffix names the double after the
		// interface it doubles, and the two share a package — so the
		// generated file would not compile, and the failure would point at
		// generated code rather than at the config line that caused it.
		double, _ := split(t, generate(t, withSuffix(t, ""), storeFixture(t)))
		testkit.Equal(t, double.TypeName, "Store"+stub.DefaultSuffix, "empty suffix falls back")
	})

	t.Run("reports a failed append rather than emitting nothing", func(t *testing.T) {
		t.Parallel()
		// Swallowing this reads downstream as an interface nobody annotated
		// rather than as a fault. Freezing the emit view provokes the failure
		// from outside the pipeline, standing in for the real cause — an
		// append arriving after Layout has closed the graph.
		s := storeFixture(t)
		s.Emit().Freeze()

		err := stub.New().Generate(generatorContext(s, sdk.NewSink()))

		testkit.Error(t, err, "a failed append must surface")
		// Which declaration the run was on, not merely that something failed.
		// The wrapping exists so a reader can find the source line, and an
		// assertion that stops at "an error happened" passes just as well
		// against one that names nothing.
		testkit.Contains(t, err.Error(), "Store", "the error must name the declaration")
	})

	t.Run("names the plugin in a failed append so the fault is attributable", func(t *testing.T) {
		t.Parallel()
		s := storeFixture(t)
		s.Emit().Freeze()

		err := stub.New().Generate(generatorContext(s, sdk.NewSink()))

		testkit.Contains(t, err.Error(), stub.Name, "the error must name the plugin")
	})
}

// The dispatch body branches on these, and getting either wrong produces
// generated code that does not compile rather than a test failure — so they
// are pinned here rather than left to the template.
func TestMethodPredicates(t *testing.T) {
	t.Parallel()

	byName := func(t *testing.T, name string) stub.Method {
		t.Helper()
		double, _ := split(t, generate(t, stub.New(), storeFixture(t)))
		for _, m := range double.Methods {
			if m.Name == name {
				return m
			}
		}
		t.Fatalf("fixture has no method %q", name)
		return stub.Method{}
	}

	t.Run("reports results on a method that returns", func(t *testing.T) {
		t.Parallel()
		testkit.True(t, byName(t, "Get").HasResults(), "Get returns a value and an error")
	})

	t.Run("reports no results on a void method", func(t *testing.T) {
		t.Parallel()
		testkit.False(t, byName(t, "Close").HasResults(), "Close returns nothing")
	})

	t.Run("finds the error slot by type rather than by position", func(t *testing.T) {
		t.Parallel()
		// Fault injection stamps the injected error onto this slot, so a
		// signature whose error is not last must still resolve correctly.
		got := byName(t, "Get").ErrReturn()
		testkit.Assert(t, got).IsNotNil("Get has an error return")
		testkit.Equal(t, got.Field, "Err", "the error slot carries its declared name")
	})

	t.Run("reports no error slot on a method that cannot fail", func(t *testing.T) {
		t.Parallel()
		testkit.Assert(t, byName(t, "Close").ErrReturn()).IsNil("Close returns no error")
	})
}

// The companion's reference to the double follows wherever Layout routed the
// primary output, which is the one fact Generate cannot know.
func TestSetOutputPackages(t *testing.T) {
	t.Parallel()

	const routed = "example.com/demo/testkit"

	t.Run("repoints the double reference at the routed package", func(t *testing.T) {
		t.Parallel()
		_, tests := split(t, generate(t, stub.New(), storeFixture(t)))
		tests.SetOutputPackages(map[string]string{"": routed})
		testkit.Equal(t, tests.StubRef.Pkg, routed, "the reference follows the primary output")
	})

	t.Run("keeps the double's name when repointing it", func(t *testing.T) {
		t.Parallel()
		_, tests := split(t, generate(t, stub.New(), storeFixture(t)))
		tests.SetOutputPackages(map[string]string{"": routed})
		testkit.Equal(t, tests.StubRef.Name, "StoreStub", "only the package moves")
	})

	t.Run("leaves the interface reference alone", func(t *testing.T) {
		t.Parallel()
		// The source interface is hand-written and does not move with the
		// generator's output. Repointing it would break exactly the case
		// redirection exists to fix.
		_, tests := split(t, generate(t, stub.New(), storeFixture(t)))
		before := tests.IfaceRef.Pkg

		tests.SetOutputPackages(map[string]string{"": routed})

		testkit.Equal(t, tests.IfaceRef.Pkg, before, "the source interface does not move")
	})

	t.Run("keeps the provisional reference when the path is underivable", func(t *testing.T) {
		t.Parallel()
		// Centralised routing resolves a Target without an import path. A
		// wrong package is a compile error naming the symbol; a bare name
		// silently binds to whatever else is in scope, so the provisional
		// value is the safer residue.
		_, tests := split(t, generate(t, stub.New(), storeFixture(t)))
		before := tests.StubRef.Pkg

		tests.SetOutputPackages(map[string]string{"": ""})

		testkit.Equal(t, tests.StubRef.Pkg, before, "an underivable path changes nothing")
	})
}

// fixturePkg is the import path every multi-interface fixture declares.
//
// Named rather than repeated because an embed has to carry it: the Go frontend
// records the declaring package on every named reference, in-package ones
// included, and [sdk.StoreReader.MethodSet] resolves an embed by qualified name.
// A fixture spelling a bare `Base` describes source no frontend produces, and
// the method set it resolves to is short by everything the embed contributes.
const fixturePkg = "example.com/storepkg"

// storeFixture returns a store holding one `//testkit:stub` interface with a
// named-return method, an unnamed-return method, and a void method — the
// three shapes the signature rules discriminate on.
func storeFixture(t *testing.T) *sdk.Store {
	t.Helper()
	return storefixture.New().
		Interface("Store", func(i *storefixture.InterfaceBuilder) {
			i.Directive(storefixture.Directive("stub"))
			i.Method("Get", func(m *storefixture.MethodBuilder) {
				gentest.Ctx(m)
				m.Param("id", storefixture.Named("string"))
				m.NamedReturn("item", storefixture.Named("string"))
				m.NamedReturn("err", storefixture.Named("error"))
			})
			i.Method("List", func(m *storefixture.MethodBuilder) {
				m.Return(storefixture.Slice(storefixture.Named("string")))
				gentest.Err(m)
			})
			i.Method("Close", nil)
		}).
		Build()
}

// mixedFixture returns a store holding an annotated interface with one
// ordinary method and one stamped integration-only, so the projection has
// something to drop and something to keep.
func mixedFixture(t *testing.T) *sdk.Store {
	t.Helper()
	return storefixture.New().
		Interface("Store", func(i *storefixture.InterfaceBuilder) {
			i.Directive(storefixture.Directive("stub"))
			i.Method("Get", func(m *storefixture.MethodBuilder) {
				m.Param("id", storefixture.Named("string"))
				gentest.Err(m)
			})
			i.Method("Connect", func(m *storefixture.MethodBuilder) {
				gentest.Err(m)
				stampMixin(m, "integrationonly")
			})
		}).
		Build()
}

// emptyFixture returns a store whose annotated interface declares no methods,
// which is the one shape a double has nothing to stand in for.
func emptyFixture() *sdk.Store {
	return storefixture.New().
		Interface("Store", func(i *storefixture.InterfaceBuilder) {
			i.Directive(storefixture.Directive("stub"))
		}).
		Build()
}

// orderedFixture returns a store whose Commit method may only follow Prepare,
// stamped the way the orderafter mixin does.
func orderedFixture() *sdk.Store {
	return storefixture.New().
		Interface("Store", func(i *storefixture.InterfaceBuilder) {
			i.Directive(storefixture.Directive("stub"))
			i.Method("Prepare", func(m *storefixture.MethodBuilder) {
				gentest.Err(m)
			})
			i.Method("Commit", func(m *storefixture.MethodBuilder) {
				gentest.Err(m)
				stampMixin(m, stub.MixinOrderAfter)
				shape.MixinParamKey(stub.MixinOrderAfter, stub.MixinOrderAfterParam).
					Set(m.Node().EnsureMeta(), "Prepare", "test")
			})
		}).
		Build()
}

// stampMixin attaches a mixin to a method the way the shape annotator would.
//
// The fixture stamps meta directly rather than running the annotator: what is
// under test here is how the generator reads a stamp, not whether the
// annotator produces one — the corpus gate covers that.
func stampMixin(m *storefixture.MethodBuilder, name string) {
	shape.MetaMixins.Set(m.Node().EnsureMeta(), []string{name}, "test")
}

// emptyInterfaceFixture returns a store holding an annotated interface that
// declares no methods — the shape the plugin diagnoses rather than emits.
func emptyInterfaceFixture() *sdk.Store {
	return storefixture.New().
		Interface("Empty", func(i *storefixture.InterfaceBuilder) {
			i.Directive(storefixture.Directive("stub"))
		}).
		Build()
}

// generatorContext wires a store and a diagnostic sink into the context the
// generator phase would hand the plugin.
func generatorContext(s *sdk.Store, d *sdk.Sink) *sdk.GeneratorContext {
	return &sdk.GeneratorContext{Store: s, Reader: sdk.NewStoreReader(s), Diag: d}
}

// withSuffix returns a plugin configured with the given suffix option.
func withSuffix(t *testing.T, suffix string) *stub.Plugin {
	t.Helper()
	p := stub.New()
	if err := p.SetOptions(sdk.NewOptions(p.OptionsSchema(), map[string]string{"suffix": suffix})); err != nil {
		t.Fatalf("SetOptions(%q): %v", suffix, err)
	}
	return p
}

// generate drives p over s and returns the queued contributions.
func generate(t *testing.T, p *stub.Plugin, s *sdk.Store) []sdk.PendingOriginSlot {
	t.Helper()
	if err := p.Generate(generatorContext(s, sdk.NewSink())); err != nil {
		t.Fatalf("Generate: %v", err)
	}
	return s.Emit().PendingOriginSlots()
}

// generateDiagnostics drives p over s and returns what it reported, for the
// cases where the diagnostic is the behaviour under test.
func generateDiagnostics(t *testing.T, p *stub.Plugin, s *sdk.Store) []sdk.Diag {
	t.Helper()
	sink := sdk.NewSink()
	if err := p.Generate(generatorContext(s, sink)); err != nil {
		t.Fatalf("Generate: %v", err)
	}
	return sink.Diagnostics()
}

// split separates the queued contributions into the primary double and the
// tagged companion, failing when either is absent.
func split(t *testing.T, pending []sdk.PendingOriginSlot) (*stub.Stub, *stub.Tests) {
	t.Helper()
	var (
		double *stub.Stub
		tests  *stub.Tests
	)
	for _, p := range pending {
		switch v := p.Item.(type) {
		case *stub.Stub:
			double = v
		case *stub.Tests:
			tests = v
		}
	}
	if double == nil || tests == nil {
		t.Fatalf("expected one double and one companion contribution; got %d", len(pending))
	}
	return double, tests
}

// A generic double's checks live in a generic helper that a concrete entry
// point instantiates, so the witnesses are what decides whether a companion
// can be written at all.
func TestWitnesses(t *testing.T) {
	t.Parallel()

	t.Run("derives a witness for a comparable parameter", func(t *testing.T) {
		t.Parallel()
		_, tests := split(t, generate(t, stub.New(), genericFixture(t, "comparable", "any", nil)))
		testkit.Len(t, tests.Witnesses, 2, "both parameters take a derived witness")
	})

	t.Run("derives distinct witnesses per position", func(t *testing.T) {
		t.Parallel()
		// Identical witnesses would let a template that crossed two type
		// parameters typecheck, which is the mistake most worth catching and
		// the one an assertion cannot see.
		_, tests := split(t, generate(t, stub.New(), genericFixture(t, "comparable", "any", nil)))
		testkit.NotEqual(t, renderRef(t, tests.Witnesses[0]), renderRef(t, tests.Witnesses[1]),
			"a crossed type parameter must not typecheck")
	})

	t.Run("writes a companion for a derivable interface", func(t *testing.T) {
		t.Parallel()
		_, tests := split(t, generate(t, stub.New(), genericFixture(t, "comparable", "any", nil)))
		testkit.False(t, tests.Generic, "a derivable double gets checks rather than a note")
	})

	t.Run("declines a constraint it cannot read", func(t *testing.T) {
		t.Parallel()
		// A named constraint is a reference into a package the generator never
		// loaded. Guessing at its type set would produce a companion that fails
		// to compile for a reason the author could not have predicted.
		_, tests := split(t, generate(t, stub.New(), genericFixture(t, "Ordered", "any", nil)))
		testkit.True(t, tests.Generic, "an unreadable constraint leaves a note")
	})

	t.Run("takes the witnesses the source pinned", func(t *testing.T) {
		t.Parallel()
		pinned := map[string]string{stub.WitnessKey: "int,Score"}
		_, tests := split(t, generate(t, stub.New(), genericFixture(t, "Ordered", "any", pinned)))
		testkit.Len(t, tests.Witnesses, 2, "a pinned list makes an opaque constraint witnessable")
	})

	t.Run("qualifies a pinned witness against the source package", func(t *testing.T) {
		t.Parallel()
		// The companion lives in an external test package and reaches nothing
		// in the source package unqualified.
		pinned := map[string]string{stub.WitnessKey: "int,Score"}
		_, tests := split(t, generate(t, stub.New(), genericFixture(t, "Ordered", "any", pinned)))
		testkit.Equal(t, renderRef(t, tests.Witnesses[1]), "example.com/storepkg.Score",
			"a witness declared in the source package carries its package")
	})

	t.Run("renders a predeclared witness bare", func(t *testing.T) {
		t.Parallel()
		pinned := map[string]string{stub.WitnessKey: "int,Score"}
		_, tests := split(t, generate(t, stub.New(), genericFixture(t, "Ordered", "any", pinned)))
		testkit.Equal(t, renderRef(t, tests.Witnesses[0]), "int",
			"a predeclared type needs no qualifier")
	})

	t.Run("leaves a non-generic double no witnesses", func(t *testing.T) {
		t.Parallel()
		_, tests := split(t, generate(t, stub.New(), storeFixture(t)))
		testkit.Len(t, tests.Witnesses, 0, "an unparameterised double instantiates nothing")
	})
}

// A witness list that does not match the type-parameter list cannot be
// positionally assigned, and guessing which parameter a lone entry meant would
// produce a compile error in generated code.
func TestWitnessCountMismatch(t *testing.T) {
	t.Parallel()

	t.Run("reports a list that is too short", func(t *testing.T) {
		t.Parallel()
		pinned := map[string]string{stub.WitnessKey: "int"}
		diags := generateDiagnostics(t, stub.New(), genericFixture(t, "Ordered", "any", pinned))
		testkit.Len(t, diags, 1, "a mismatched witness list is reported once")
	})

	t.Run("names the count it was given", func(t *testing.T) {
		t.Parallel()
		pinned := map[string]string{stub.WitnessKey: "int"}
		diags := generateDiagnostics(t, stub.New(), genericFixture(t, "Ordered", "any", pinned))
		testkit.Contains(t, diags[0].Message, "1 type for 2 type parameters",
			"the diagnostic names both counts")
	})

	t.Run("does not fall back to a guess", func(t *testing.T) {
		t.Parallel()
		// Falling through to derivation would replace a stated intent with a
		// guess, and the author would never learn their line was ignored.
		pinned := map[string]string{stub.WitnessKey: "int"}
		_, tests := split(t, generate(t, stub.New(), genericFixture(t, "Ordered", "any", pinned)))
		testkit.Len(t, tests.Witnesses, 0, "a rejected list is not silently replaced")
	})
}

// genericFixture returns a two-parameter generic store, with each constraint
// named and an optional set of directive keys.
//
// A constraint named `any` or `comparable` is spelled as the frontend spells
// it — an embedded bound plus the printed source form — because that is the
// shape the derivation reads, and a fixture that carried only one of the two
// would pass against a projection that read the wrong one.
func genericFixture(t *testing.T, kBound, vBound string, kv map[string]string) *sdk.Store {
	t.Helper()
	dir := storefixture.Directive("stub")
	maps.Copy(dir.KV, kv)
	// A routing directive sits alongside the plugin's own on every real
	// interface, so the witness read has to pass over one it does not own.
	routing := storefixture.Directive("out", storefixture.Arg("storetest/"))
	return storefixture.New().
		Package("storepkg", fixturePkg).
		Interface("Store", func(i *storefixture.InterfaceBuilder) {
			i.Directive(routing)
			i.Directive(dir)
			i.TypeParam("K", bound(kBound))
			i.TypeParam("V", bound(vBound))
			i.Method("Get", func(m *storefixture.MethodBuilder) {
				m.Param("key", storefixture.Named("K"))
				m.Return(storefixture.Named("V"))
				gentest.Err(m)
			})
		}).
		Build()
}

// bound builds the constraint the frontend produces for a written bound.
func bound(name string) *sdk.Constraint {
	c := storefixture.Constraint(storefixture.Named(name))
	c.Raw = name
	return c
}

// renderRef spells a witness the way the backend would, which is the only form
// a reader of the generated file sees.
func renderRef(t *testing.T, r sdk.Ref) string {
	t.Helper()
	switch typed := r.(type) {
	case *sdk.BuiltinRef:
		return typed.Name
	case *sdk.ExternalRef:
		return typed.Package + "." + typed.Name
	default:
		t.Fatalf("witness is %T, want a builtin or external reference", r)
		return ""
	}
}

// An interface's method set includes whatever it embeds, and a double missing
// one of those methods does not satisfy the interface it doubles — so what is
// under test here is a resolution failure producing no artefact rather than a
// partial one.
func TestMethodSet(t *testing.T) {
	t.Parallel()

	t.Run("carries an embedded interface's methods", func(t *testing.T) {
		t.Parallel()
		double, _ := split(t, generate(t, stub.New(), embeddedFixture(t, "Base")))
		testkit.Len(t, double.Methods, 2, "the double carries the embedded method and its own")
	})

	t.Run("orders declared methods before embedded ones", func(t *testing.T) {
		t.Parallel()
		// The order [sdk.MethodSet] documents: the interface's own
		// declarations, then each embed's contribution in embed order. What
		// matters to the generated fields is that the order is fixed, so they
		// stay put as an embedded interface gains a method.
		double, _ := split(t, generate(t, stub.New(), embeddedFixture(t, "Base")))
		testkit.Equal(t, double.Methods[0].Name, "Get", "the declared method comes first")
		testkit.Equal(t, double.Methods[1].Name, "Ping", "the embed contributes after it")
	})

	t.Run("attributes an embedded method to the interface that carried it", func(t *testing.T) {
		t.Parallel()
		double, _ := split(t, generate(t, stub.New(), embeddedFixture(t, "Base")))
		testkit.Equal(t, double.Methods[0].From, "", "a declared method is attributed to nothing")
		testkit.Equal(t, double.Methods[1].From, "Base", "the generated field says where it came from")
	})

	t.Run("emits nothing when an embed cannot be resolved", func(t *testing.T) {
		t.Parallel()
		// A double missing a method cannot be passed anywhere the interface is
		// expected, so there is no useful partial result to emit.
		pending := generate(t, stub.New(), embeddedFixture(t, "Missing"))
		testkit.Len(t, pending, 0, "an unresolvable embed produces no double")
	})

	t.Run("warns with the embed spelled as the source wrote it", func(t *testing.T) {
		t.Parallel()
		diags := generateDiagnostics(t, stub.New(), embeddedFixture(t, "Missing"))
		testkit.Len(t, diags, 1, "an unresolvable embed is reported once")
		testkit.Contains(t, diags[0].Message, "Missing", "the diagnostic names the embed")
	})

	t.Run("does not fail the run for an unresolvable embed", func(t *testing.T) {
		t.Parallel()
		// Every other interface in the same run still generates, so one
		// unreachable dependency does not cost a project its doubles.
		diags := generateDiagnostics(t, stub.New(), embeddedFixture(t, "Missing"))
		testkit.Equal(t, diags[0].Severity, sdk.SeverityWarn,
			"an unreachable embed is a warning, not a failure")
	})
}

// A diagnostic naming a bare `Closer` sends the author looking in the wrong
// package, so an embed is spelled the way the source wrote it.
func TestMethodSetForeignEmbed(t *testing.T) {
	t.Parallel()

	t.Run("qualifies an embed from another package", func(t *testing.T) {
		t.Parallel()
		diags := generateDiagnostics(t, stub.New(), foreignEmbedFixture(t))
		testkit.Len(t, diags, 1, "an unreachable embed is reported once")
		testkit.Contains(t, diags[0].Message, `"io.Closer"`,
			"the diagnostic spells the embed as the source wrote it")
	})

	t.Run("skips a union term carrying no name", func(t *testing.T) {
		t.Parallel()
		// A type-set term sits in the same list as an embedded interface and
		// contributes no methods. Such a type is never a stub target, so it is
		// passed over rather than reported.
		diags := generateDiagnostics(t, stub.New(), unionTermFixture(t))
		testkit.Len(t, diags, 0, "a type-set term is not an unresolved embed")
	})
}

// foreignEmbedFixture returns an interface embedding one from another package.
func foreignEmbedFixture(t *testing.T) *sdk.Store {
	t.Helper()
	return storefixture.New().
		Package("storepkg", fixturePkg).
		Interface("Stream", func(i *storefixture.InterfaceBuilder) {
			i.Directive(storefixture.Directive("stub"))
			i.Embed(storefixture.PkgNamed("io", "Closer"))
			i.Method("Read", func(m *storefixture.MethodBuilder) {
				gentest.Err(m)
			})
		}).
		Build()
}

// unionTermFixture returns an interface whose embed list holds a term with no
// name, as a type set does.
func unionTermFixture(t *testing.T) *sdk.Store {
	t.Helper()
	return storefixture.New().
		Package("storepkg", fixturePkg).
		Interface("Termed", func(i *storefixture.InterfaceBuilder) {
			i.Directive(storefixture.Directive("stub"))
			i.Embed(storefixture.Slice(storefixture.Named("byte")))
			i.Method("Get", func(m *storefixture.MethodBuilder) {
				gentest.Err(m)
			})
		}).
		Build()
}

// The embed walk has three arms nothing in the corpus reaches, each of which
// would corrupt a method set rather than fail loudly if it were wrong.
func TestMethodSetEdges(t *testing.T) {
	t.Parallel()

	t.Run("terminates on a cyclic embed graph", func(t *testing.T) {
		t.Parallel()
		// Illegal in Go and unreachable from a real frontend, so the guard is
		// checked against a hand-built graph — a malformed one should cost a
		// diagnostic rather than the process. The walk breaks the cycle only
		// after the interface it points back at has contributed, so the set is
		// short of nothing and the double is still worth emitting.
		pending := generate(t, stub.New(), cyclicFixture(t))
		testkit.Len(t, pending, 2, "a cycle still yields the double and its companion")
	})

	t.Run("reports the cycle it broke out of", func(t *testing.T) {
		t.Parallel()
		// Emitting anyway is not the same as saying nothing: the source is
		// illegal Go, and a run that generated a working double from it without
		// a word would leave the author to find that out from the compiler.
		diags := generateDiagnostics(t, stub.New(), cyclicFixture(t))
		testkit.Len(t, diags, 1, "a cycle is reported once")
		testkit.Equal(t, diags[0].Severity, sdk.SeverityWarn,
			"a cycle costs no method, so it is a warning")
	})

	t.Run("refuses an embed that resolves to something other than an interface", func(t *testing.T) {
		t.Parallel()
		// Distinct from an embed this run did not load: no wider run repairs a
		// struct in an embed list, so the author must be sent to their source
		// rather than to their -target filter.
		diags := generateDiagnostics(t, stub.New(), structEmbedFixture(t))
		testkit.Len(t, diags, 1, "a non-interface embed is reported once")
		testkit.Equal(t, diags[0].Severity, sdk.SeverityError,
			"a source defect is an error, not a warning")
	})

	t.Run("carries an unresolved embed up through a nested one", func(t *testing.T) {
		t.Parallel()
		// The failure is two levels down; a walk that only reported its own
		// level would emit a double missing a method it never knew about.
		pending := generate(t, stub.New(), nestedMissingFixture(t))
		testkit.Len(t, pending, 0, "an embed's own unresolved embed blocks the double")
	})

	t.Run("takes an overlapping method once", func(t *testing.T) {
		t.Parallel()
		// Go admits two embeds declaring the same method only where the
		// signatures agree, so the second arrival is the same method and a
		// double declaring it twice would not compile.
		double, _ := split(t, generate(t, stub.New(), overlappingFixture(t)))
		testkit.Len(t, double.Methods, 1, "an overlapping method is carried once")
	})
}

// cyclicFixture returns two interfaces embedding each other.
func cyclicFixture(t *testing.T) *sdk.Store {
	t.Helper()
	return storefixture.New().
		Package("storepkg", fixturePkg).
		Interface("A", func(i *storefixture.InterfaceBuilder) {
			i.Directive(storefixture.Directive("stub"))
			i.Embed(storefixture.PkgNamed(fixturePkg, "B"))
			i.Method("Get", func(m *storefixture.MethodBuilder) {
				gentest.Err(m)
			})
		}).
		Interface("B", func(i *storefixture.InterfaceBuilder) {
			i.Embed(storefixture.PkgNamed(fixturePkg, "A"))
		}).
		Build()
}

// structEmbedFixture returns an interface embedding a struct, which resolves
// to a declaration that cannot carry a method set.
func structEmbedFixture(t *testing.T) *sdk.Store {
	t.Helper()
	return storefixture.New().
		Package("storepkg", fixturePkg).
		Struct("Config", nil).
		Interface("Broken", func(i *storefixture.InterfaceBuilder) {
			i.Directive(storefixture.Directive("stub"))
			i.Embed(storefixture.PkgNamed(fixturePkg, "Config"))
			i.Method("Get", func(m *storefixture.MethodBuilder) {
				gentest.Err(m)
			})
		}).
		Build()
}

// nestedMissingFixture returns an interface whose embed embeds something the
// store does not hold.
func nestedMissingFixture(t *testing.T) *sdk.Store {
	t.Helper()
	return storefixture.New().
		Package("storepkg", fixturePkg).
		Interface("Middle", func(i *storefixture.InterfaceBuilder) {
			i.Embed(storefixture.PkgNamed(fixturePkg, "Missing"))
			i.Method("Ping", func(m *storefixture.MethodBuilder) {
				gentest.Err(m)
			})
		}).
		Interface("Outer", func(i *storefixture.InterfaceBuilder) {
			i.Directive(storefixture.Directive("stub"))
			i.Embed(storefixture.PkgNamed(fixturePkg, "Middle"))
			i.Method("Get", func(m *storefixture.MethodBuilder) {
				gentest.Err(m)
			})
		}).
		Build()
}

// overlappingFixture returns an interface embedding two that declare the same
// method.
func overlappingFixture(t *testing.T) *sdk.Store {
	t.Helper()
	shared := func(i *storefixture.InterfaceBuilder) {
		i.Method("Close", func(m *storefixture.MethodBuilder) {
			gentest.Err(m)
		})
	}
	return storefixture.New().
		Package("storepkg", fixturePkg).
		Interface("Left", shared).
		Interface("Right", shared).
		Interface("Both", func(i *storefixture.InterfaceBuilder) {
			i.Directive(storefixture.Directive("stub"))
			i.Embed(storefixture.PkgNamed(fixturePkg, "Left"))
			i.Embed(storefixture.PkgNamed(fixturePkg, "Right"))
		}).
		Build()
}

// A generic embed's methods name the embedded interface's type parameters, not
// the embedder's, and resolution copies signatures without substituting them.
func TestMethodSetGenericEmbed(t *testing.T) {
	t.Parallel()

	t.Run("refuses an embed carrying type arguments", func(t *testing.T) {
		t.Parallel()
		diags := generateDiagnostics(t, stub.New(), genericEmbedFixture(t))
		testkit.Len(t, diags, 1, "a generic embed is reported once")
		testkit.Equal(t, diags[0].Severity, sdk.SeverityError,
			"a substitution the projection cannot do is an error, not a warning")
	})

	t.Run("emits nothing for it", func(t *testing.T) {
		t.Parallel()
		pending := generate(t, stub.New(), genericEmbedFixture(t))
		testkit.Len(t, pending, 0, "a refused embed produces no double")
	})
}

// The witness palette runs out before an interface's type parameters do only
// at an arity nothing sane reaches, but running off the end would index out of
// range rather than decline.
func TestWitnessPaletteExhausted(t *testing.T) {
	t.Parallel()

	t.Run("declines an interface with more parameters than witnesses", func(t *testing.T) {
		t.Parallel()
		_, tests := split(t, generate(t, stub.New(), wideGenericFixture(t)))
		testkit.True(t, tests.Generic, "an arity past the palette leaves a note")
	})
}

// Generic is what the companion branches on, and it answers for the double
// rather than for the companion that reads it.
func TestStubGeneric(t *testing.T) {
	t.Parallel()

	t.Run("reports a parameterised double", func(t *testing.T) {
		t.Parallel()
		double, _ := split(t, generate(t, stub.New(), genericFixture(t, "comparable", "any", nil)))
		testkit.True(t, double.Generic(), "a double carrying type parameters is generic")
	})

	t.Run("reports a plain double", func(t *testing.T) {
		t.Parallel()
		double, _ := split(t, generate(t, stub.New(), storeFixture(t)))
		testkit.False(t, double.Generic(), "a double with no type parameters is not")
	})
}

// genericEmbedFixture returns an interface embedding a generic one at an
// instantiation, which is the shape resolution cannot carry.
func genericEmbedFixture(t *testing.T) *sdk.Store {
	t.Helper()
	return storefixture.New().
		Package("storepkg", fixturePkg).
		Interface("Base", func(i *storefixture.InterfaceBuilder) {
			i.TypeParam("K", bound("any"))
			i.Method("Ping", func(m *storefixture.MethodBuilder) {
				gentest.Err(m)
			})
		}).
		Interface("Composed", func(i *storefixture.InterfaceBuilder) {
			i.Directive(storefixture.Directive("stub"))
			i.Embed(instantiated(fixturePkg, "Base", "string"))
			i.Method("Get", func(m *storefixture.MethodBuilder) {
				gentest.Err(m)
			})
		}).
		Build()
}

// instantiated builds a reference to a generic type at one type argument.
func instantiated(pkg, name, arg string) *sdk.TypeRef {
	ref := storefixture.PkgNamed(pkg, name)
	ref.TypeArgs = []*sdk.TypeRef{storefixture.Named(arg)}
	return ref
}

// wideGenericFixture returns an interface with more type parameters than the
// witness palette holds.
func wideGenericFixture(t *testing.T) *sdk.Store {
	t.Helper()
	return storefixture.New().
		Package("storepkg", fixturePkg).
		Interface("Wide", func(i *storefixture.InterfaceBuilder) {
			i.Directive(storefixture.Directive("stub"))
			for _, n := range []string{"A", "B", "C", "D", "E", "F", "G", "H", "I"} {
				i.TypeParam(n, bound("any"))
			}
			i.Method("Get", func(m *storefixture.MethodBuilder) {
				gentest.Err(m)
			})
		}).
		Build()
}

// embeddedFixture returns a store whose Composed interface embeds embedName
// and declares one method of its own. Naming an interface the store does not
// hold exercises the unresolvable path.
func embeddedFixture(t *testing.T, embedName string) *sdk.Store {
	t.Helper()
	return storefixture.New().
		Package("storepkg", fixturePkg).
		Interface("Base", func(i *storefixture.InterfaceBuilder) {
			i.Method("Ping", func(m *storefixture.MethodBuilder) {
				gentest.Err(m)
			})
		}).
		Interface("Composed", func(i *storefixture.InterfaceBuilder) {
			i.Directive(storefixture.Directive("stub"))
			i.Embed(storefixture.PkgNamed(fixturePkg, embedName))
			i.Method("Get", func(m *storefixture.MethodBuilder) {
				gentest.Err(m)
			})
		}).
		Build()
}

// Rendering is where a generator actually fails. Every assertion driven off
// the emit graph passes against a template that renders code which does not
// compile — an unused local, a redeclared name, a field the template still
// references. Those surface only once the backend runs, so the templates are
// driven end-to-end here rather than trusted.
func TestRender(t *testing.T) {
	t.Parallel()

	t.Run("renders both outputs", func(t *testing.T) {
		t.Parallel()
		renderFixture(t).AssertFileCount(2)
	})

	t.Run("renders without a diagnostic", func(t *testing.T) {
		t.Parallel()
		testkit.Len(t, renderFixture(t).Diagnostics().Diagnostics(), 0,
			"a clean fixture must render without diagnostics")
	})

	t.Run("emits the double's configuration surface", func(t *testing.T) {
		t.Parallel()
		// Asserted on tokens rather than whole lines: the backend runs the
		// formatter, so alignment is padding this test has no business
		// pinning. The golden below covers the exact bytes.
		f := renderFixture(t).AssertFile(primaryFile)
		for _, want := range []string{
			"type StoreGetStub struct",
			"*stub.MethodStub[StoreGetCall]",
			"func (s *StoreGetStub) Returns(item string, err error) *StoreGetStub",
			"type StoreGetReturn struct",
			"type StoreStub struct",
			"OnGet   *StoreGetStub",
			"func NewStoreStub(tb testing.TB, opts ...StoreStubOption) *StoreStub",
			"func (s *StoreStub) ResetCalls()",
			"func StoreStubStrict() StoreStubOption",
			"func StoreStubDelegateTo(",
			"func StoreStubWithClock(",
			"func StoreStubBenchMode() StoreStubOption",
			"func WithStoreGet(",
		} {
			f.Contains(want)
		}
	})

	t.Run("resolves a call through the shared dispatcher", func(t *testing.T) {
		t.Parallel()
		// Which arm answers, and that every arm records, is stub.Answer's
		// contract and is tested there against real calls. What the template
		// owes is the binding: the arms wired to the right fields, so a
		// double cannot dispatch differently from every other double.
		f := renderFixture(t).AssertFile(primaryFile)
		for _, want := range []string{
			"r := stub.Answer(s.OnGet.MethodStub, &call, stub.Arms[StoreGetCall, StoreGetReturn]{",
			"Invoke:   s.OnGet.invoke(ctx, id),",
			"Fallback: s.OnGet.fallback,",
			"Fault:    func(err error) StoreGetReturn { return StoreGetReturn{Err: err} },",
		} {
			f.Contains(want)
		}
	})

	t.Run("gives a method that cannot fail no fault arm", func(t *testing.T) {
		t.Parallel()
		// An injected fault has nowhere to go in a signature with no error,
		// and a nil Fault is how Answer knows to skip that arm rather than
		// inventing an error slot.
		//
		// Counted rather than sliced out of the body: the fixture's Get and
		// List return errors and Close does not, so a fault arm appearing for
		// Close shows up as a third occurrence.
		body := renderFixture(t).AssertFile(primaryFile).String()
		testkit.Equal(t, strings.Count(body, "Fault:    func(err error)"), 2,
			"only the two methods that can fail declare a fault arm")
	})

	t.Run("types a void method's arms with the empty return tuple", func(t *testing.T) {
		t.Parallel()
		// A method returning nothing still has a call worth recording and an
		// override worth dispatching to; the empty struct is the return tuple
		// of no returns.
		renderFixture(t).AssertFile(primaryFile).
			Contains("stub.Arms[StoreCloseCall, struct{}]{")
	})

	t.Run("puts the companion in the external test package", func(t *testing.T) {
		t.Parallel()
		renderFixture(t).AssertFile(companionFile).Contains("package storepkg_test")
	})
}

// The goldens are the readable record of what this generator produces. A diff
// on them is the review surface for any template change — the token
// assertions above say a construct is present, and only the golden says what
// the whole file reads like.
//
// Regenerate with `go test ./generator/stub/ -update-golden`.
func TestRenderMatchesGolden(t *testing.T) {
	t.Parallel()

	t.Run("the double", func(t *testing.T) {
		t.Parallel()
		matchesGolden(t, primaryFile)
	})

	t.Run("the companion", func(t *testing.T) {
		t.Parallel()
		matchesGolden(t, companionFile)
	})
}

// matchesGolden compares one rendered file against its golden, with the
// header's Command line normalised.
//
// The backend stamps os.Args into that line when nothing sets it, which under
// `go test` is the test binary's own flags — including a temp path that
// differs on every run. Normalising it keeps the golden about the generated
// code rather than about how the suite was invoked.
func matchesGolden(t *testing.T, name string) {
	t.Helper()
	body := renderFixture(t).AssertFile(name).String()
	pipelinetest.MatchesGoldenBytes(t, []byte(normaliseCommand(body)), "testdata/golden/"+name)
}

// normaliseCommand rewrites the header's Command line to a fixed value.
func normaliseCommand(body string) string {
	lines := strings.Split(body, "\n")
	for i, line := range lines {
		if strings.HasPrefix(line, "// Command:") {
			lines[i] = "// Command:   testkit run ./storepkg"
			break
		}
	}
	return strings.Join(lines, "\n")
}

// The rendered filenames, composed by Layout from the source basename and the
// adapter's suffixes.
const (
	primaryFile   = "store" + stub.GoPrimarySuffix
	companionFile = "store" + stub.GoTestSuffix
)

// renderFixture drives the plugin and the Go backend over the shared fixture
// through a synthetic pipeline, so routing and rendering both participate.
func renderFixture(t *testing.T) *pipelinetest.Pipeline {
	t.Helper()
	return pipelinetest.New(t).
		WithFrontend(pipelinetest.FromNodes(fixturePackage())).
		WithGenerator(stub.New()).
		WithBackend(backendgolang.New()).
		Build().
		Run()
}

// fixturePackage is storeFixture's package node, for the synthetic frontend.
func fixturePackage() *sdk.Package { return fixtureBuilder().PackageNode() }

// fixtureBuilder is the fixture itself, kept as the builder rather than only
// its package node so [Builder.GoSource] can project the hand-written package
// the generated double references.
//
// That projection is what makes the toolchain assertions mean anything. A
// double is checked against the interface it doubles, and supplying that
// interface as a separately maintained support file makes the fixture that
// drove the run and the source it stands for two things that can disagree —
// silently, because a stale support file still compiles.
func fixtureBuilder() *storefixture.Builder {
	return storefixture.New().
		Package("storepkg", fixturePkg).
		Interface("Store", func(i *storefixture.InterfaceBuilder) {
			// Layout composes the output filename from the source basename,
			// so the fixture needs a position for the rendered names to be
			// anything other than a bare suffix.
			i.Pos(gentest.AtFile("storepkg/store.go"))
			i.Directive(storefixture.Directive("stub"))
			i.Method("Get", func(m *storefixture.MethodBuilder) {
				gentest.Ctx(m)
				m.Param("id", storefixture.Named("string"))
				m.NamedReturn("item", storefixture.Named("string"))
				m.NamedReturn("err", storefixture.Named("error"))
			})
			i.Method("List", func(m *storefixture.MethodBuilder) {
				m.Return(storefixture.Slice(storefixture.Named("string")))
				gentest.Err(m)
			})
			i.Method("Close", nil)
		})
}

// What the double is FOR is standing in for the interface, and nothing above
// this point checks that it does.
//
// Every structural assertion in this file passes against a double that does not
// satisfy Store: a dropped variadic marker, a method lost through an embed, or a
// return whose type shifted all compile perfectly and satisfy nothing. Only the
// compiler can say otherwise, so these four assertions are the ones that make
// the rest of the suite mean something.
//
// The toolchain is shelled out to once per assertion at roughly a second each,
// so the module is built once here and the four share it rather than each
// standing up its own.
func TestGeneratedCodeHoldsUp(t *testing.T) {
	t.Parallel()

	gen := generatedModule(t)

	t.Run("compiles against the package it doubles", func(t *testing.T) {
		t.Parallel()
		gen.AssertCompiles(t)
	})

	t.Run("passes go vet", func(t *testing.T) {
		t.Parallel()
		// Catches what compiling does not: an unused local, a shadowed
		// variable, a Printf verb that does not match its argument.
		gen.AssertVets(t)
	})

	t.Run("satisfies the interface it doubles", func(t *testing.T) {
		t.Parallel()
		// The whole contract, in one line the compiler decides.
		gen.AssertSatisfies(t, "StoreStub", "Store")
	})

	t.Run("the companion it emits passes", func(t *testing.T) {
		t.Parallel()
		// This generator emits a test suite, and until now nothing ran it. A
		// generated check that cannot fail is worse than no check: it reports
		// the double as verified having asserted nothing.
		gen.AssertTestsPass(t)
	})
}

func generatedModule(t *testing.T) *golangtest.Generated {
	t.Helper()
	b := fixtureBuilder()
	// A double imports the testkit runtime — clock, rand and stub — so a module
	// holding only the generated files cannot resolve them, and the sandbox
	// runs with GOPROXY=off. WithRequire wires the checkout in and raises the
	// throwaway module's go directive to the runtime's own floor, which is what
	// keeps the assertion honest: it builds against the runtime in this tree,
	// not a published one that may differ.
	return golangtest.Render(t, backendgolang.New(), b.PackageNode(), stub.New()).
		WithSource(golangtest.GoFile(b.GoSource())).
		WithRequire(stub.Module, filepath.Join("..", ".."))
}

// A constraint interface carries a type set rather than a method-set contract,
// so there is nothing for a double to stand in for.
//
// Declined by reading the Go frontend's own stamp, which is the only answer
// that knows: a term like `~MyInt` is a Named reference and indistinguishable
// by shape from an interface the run did not load, so leaving it to the embed
// walk reports it as an unresolved embed — naming a type the author never
// wrote.
func TestGenerateDeclinesAConstraint(t *testing.T) {
	t.Parallel()

	t.Run("emits nothing for a constraint", func(t *testing.T) {
		t.Parallel()
		testkit.Assert(t, generate(t, stub.New(), constraintFixture(t))).
			IsEmpty("a type set has no behaviour to double")
	})

	t.Run("says why rather than complaining about an embed", func(t *testing.T) {
		t.Parallel()
		// The failure this replaces reported "embeds %q, which was not loaded"
		// against `~int` — a term the author wrote as a type set, not as an
		// embed, and which no wider run would ever resolve.
		got := plugintest.Generate(t, stub.New(), constraintFixture(t)).Diagnostics()
		testkit.Len(t, got, 1, "a declined constraint is reported once")
		testkit.Contains(t, got[0].Message, "generic constraint",
			"the diagnostic names what the declaration actually is")
	})
}

// constraintFixture is `interface{ ~int | ~float64 }` carrying the directive,
// stamped the way the Go frontend stamps one.
func constraintFixture(t *testing.T) *sdk.Store {
	t.Helper()
	b := storefixture.New().
		Package("storepkg", fixturePkg).
		Interface("Numeric", func(i *storefixture.InterfaceBuilder) {
			i.Pos(gentest.AtFile("storepkg/store.go"))
			i.Directive(storefixture.Directive("stub"))
			i.Embed(storefixture.Named("int"))
			i.Embed(storefixture.Named("float64"))
		})
	for _, iface := range b.PackageNode().Interfaces {
		golang.MetaIsConstraintInterface.Set(iface.EnsureMeta(), true, "test")
	}
	return b.Build()
}
