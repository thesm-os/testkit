// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package stub_test

import (
	"maps"
	"testing"

	"go.thesmos.sh/eidos/core/diag"
	"go.thesmos.sh/eidos/core/opt"
	"go.thesmos.sh/eidos/eidostest/plugintest"
	"go.thesmos.sh/eidos/eidostest/storefixture"
	"go.thesmos.sh/eidos/emit"
	"go.thesmos.sh/eidos/node"
	"go.thesmos.sh/eidos/plugin"
	"go.thesmos.sh/eidos/plugins/annotator/shape"
	"go.thesmos.sh/eidos/store"

	"go.thesmos.sh/testkit"
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
				BuildStore: func(t *testing.T) *store.Store { t.Helper(); return storeFixture(t) },
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
		d := diag.New()

		_ = stub.New().Generate(generatorContext(s, d))

		testkit.True(t, d.HasErrors(), "an interface with no methods must be diagnosed")
	})

	t.Run("continues the run after reporting an empty interface", func(t *testing.T) {
		t.Parallel()
		// A diagnosed fixture is skipped, not fatal: one bad interface must
		// not stop every other interface in the run from generating.
		err := stub.New().Generate(generatorContext(emptyInterfaceFixture(), diag.New()))

		testkit.NoError(t, err, "a reported interface must not abort the run")
	})

	t.Run("queues nothing for an annotated interface with no methods", func(t *testing.T) {
		t.Parallel()
		s := emptyInterfaceFixture()

		_ = stub.New().Generate(generatorContext(s, diag.New()))

		testkit.Assert(t, s.Emit().PendingOriginSlots()).IsEmpty("a diagnosed interface emits nothing")
	})

	t.Run("drops an integration-only method from the double", func(t *testing.T) {
		t.Parallel()
		// The method reaches a real dependency, so a recorded stand-in would
		// answer for something it never did.
		double, _ := split(t, generate(t, stub.New(), mixedFixture(t)))
		for _, m := range double.Methods {
			testkit.NotEqual(t, m.Name, "Connect", "an integration-only method is not doubled")
		}
	})

	t.Run("withholds the compile-time assertion once a method is dropped", func(t *testing.T) {
		t.Parallel()
		// A double missing a method cannot satisfy its interface, so asserting
		// that it does would fail inside generated code.
		double, _ := split(t, generate(t, stub.New(), mixedFixture(t)))
		testkit.False(t, double.Complete, "a double with a dropped method is incomplete")
	})

	t.Run("reports an interface whose every method is undoubleable", func(t *testing.T) {
		t.Parallel()
		// Measured after projection: the interface has a method, but nothing
		// a double can stand in for, and an empty shell would satisfy and
		// record nothing.
		s := integrationOnlyFixture()
		d := diag.New()

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

		err := stub.New().Generate(generatorContext(s, diag.New()))

		testkit.Error(t, err, "a failed append must surface")
	})

	t.Run("names the plugin in a failed append so the fault is attributable", func(t *testing.T) {
		t.Parallel()
		s := storeFixture(t)
		s.Emit().Freeze()

		err := stub.New().Generate(generatorContext(s, diag.New()))

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

// storeFixture returns a store holding one `//testkit:stub` interface with a
// named-return method, an unnamed-return method, and a void method — the
// three shapes the signature rules discriminate on.
func storeFixture(t *testing.T) *store.Store {
	t.Helper()
	return storefixture.New().
		Interface("Store", func(i *storefixture.InterfaceBuilder) {
			i.Directive(storefixture.Directive("stub"))
			i.Method("Get", func(m *storefixture.MethodBuilder) {
				m.Param("ctx", storefixture.PkgNamed("context", "Context"))
				m.Param("id", storefixture.Named("string"))
				m.NamedReturn("item", storefixture.Named("string"))
				m.NamedReturn("err", storefixture.Named("error"))
			})
			i.Method("List", func(m *storefixture.MethodBuilder) {
				m.Return(storefixture.Slice(storefixture.Named("string")))
				m.Return(storefixture.Named("error"))
			})
			i.Method("Close", nil)
		}).
		Build()
}

// mixedFixture returns a store holding an annotated interface with one
// ordinary method and one stamped integration-only, so the projection has
// something to drop and something to keep.
func mixedFixture(t *testing.T) *store.Store {
	t.Helper()
	return storefixture.New().
		Interface("Store", func(i *storefixture.InterfaceBuilder) {
			i.Directive(storefixture.Directive("stub"))
			i.Method("Get", func(m *storefixture.MethodBuilder) {
				m.Param("id", storefixture.Named("string"))
				m.Return(storefixture.Named("error"))
			})
			i.Method("Connect", func(m *storefixture.MethodBuilder) {
				m.Return(storefixture.Named("error"))
				stampMixin(m, stub.MixinIntegrationOnly)
			})
		}).
		Build()
}

// integrationOnlyFixture returns a store whose only method is stamped
// integration-only, leaving nothing for a double to stand in for.
func integrationOnlyFixture() *store.Store {
	return storefixture.New().
		Interface("Store", func(i *storefixture.InterfaceBuilder) {
			i.Directive(storefixture.Directive("stub"))
			i.Method("Connect", func(m *storefixture.MethodBuilder) {
				m.Return(storefixture.Named("error"))
				stampMixin(m, stub.MixinIntegrationOnly)
			})
		}).
		Build()
}

// orderedFixture returns a store whose Commit method may only follow Prepare,
// stamped the way the orderafter mixin does.
func orderedFixture() *store.Store {
	return storefixture.New().
		Interface("Store", func(i *storefixture.InterfaceBuilder) {
			i.Directive(storefixture.Directive("stub"))
			i.Method("Prepare", func(m *storefixture.MethodBuilder) {
				m.Return(storefixture.Named("error"))
			})
			i.Method("Commit", func(m *storefixture.MethodBuilder) {
				m.Return(storefixture.Named("error"))
				stampMixin(m, stub.MixinOrderAfter)
				shape.MixinParamKey(stub.MixinOrderAfter, stub.MixinOrderAfterParam).
					Set(m.Node().Meta(), "Prepare", "test")
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
	shape.MetaMixins.Set(m.Node().Meta(), []string{name}, "test")
}

// emptyInterfaceFixture returns a store holding an annotated interface that
// declares no methods — the shape the plugin diagnoses rather than emits.
func emptyInterfaceFixture() *store.Store {
	return storefixture.New().
		Interface("Empty", func(i *storefixture.InterfaceBuilder) {
			i.Directive(storefixture.Directive("stub"))
		}).
		Build()
}

// generatorContext wires a store and a diagnostic sink into the context the
// generator phase would hand the plugin.
func generatorContext(s *store.Store, d *diag.Sink) *plugin.GeneratorContext {
	return &plugin.GeneratorContext{Store: s, Reader: store.NewReader(s), Diag: d}
}

// withSuffix returns a plugin configured with the given suffix option.
func withSuffix(t *testing.T, suffix string) *stub.Plugin {
	t.Helper()
	p := stub.New()
	if err := p.SetOptions(opt.New(p.OptionsSchema(), map[string]string{"suffix": suffix})); err != nil {
		t.Fatalf("SetOptions(%q): %v", suffix, err)
	}
	return p
}

// generate drives p over s and returns the queued contributions.
func generate(t *testing.T, p *stub.Plugin, s *store.Store) []store.PendingOriginSlot {
	t.Helper()
	if err := p.Generate(generatorContext(s, diag.New())); err != nil {
		t.Fatalf("Generate: %v", err)
	}
	return s.Emit().PendingOriginSlots()
}

// generateDiagnostics drives p over s and returns what it reported, for the
// cases where the diagnostic is the behaviour under test.
func generateDiagnostics(t *testing.T, p *stub.Plugin, s *store.Store) []diag.Diag {
	t.Helper()
	sink := diag.New()
	if err := p.Generate(generatorContext(s, sink)); err != nil {
		t.Fatalf("Generate: %v", err)
	}
	return sink.Diagnostics()
}

// split separates the queued contributions into the primary double and the
// tagged companion, failing when either is absent.
func split(t *testing.T, pending []store.PendingOriginSlot) (*stub.Stub, *stub.Tests) {
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
func genericFixture(t *testing.T, kBound, vBound string, kv map[string]string) *store.Store {
	t.Helper()
	dir := storefixture.Directive("stub")
	maps.Copy(dir.KV, kv)
	return storefixture.New().
		Package("storepkg", "example.com/storepkg").
		Interface("Store", func(i *storefixture.InterfaceBuilder) {
			i.Directive(dir)
			i.TypeParam("K", bound(kBound))
			i.TypeParam("V", bound(vBound))
			i.Method("Get", func(m *storefixture.MethodBuilder) {
				m.Param("key", storefixture.Named("K"))
				m.Return(storefixture.Named("V"))
				m.Return(storefixture.Named("error"))
			})
		}).
		Build()
}

// bound builds the constraint the frontend produces for a written bound.
func bound(name string) *node.Constraint {
	c := storefixture.Constraint(storefixture.Named(name))
	c.Raw = name
	return c
}

// renderRef spells a witness the way the backend would, which is the only form
// a reader of the generated file sees.
func renderRef(t *testing.T, r emit.Ref) string {
	t.Helper()
	switch typed := r.(type) {
	case *emit.BuiltinRef:
		return typed.Name
	case *emit.ExternalRef:
		return typed.Package + "." + typed.Name
	default:
		t.Fatalf("witness is %T, want a builtin or external reference", r)
		return ""
	}
}
