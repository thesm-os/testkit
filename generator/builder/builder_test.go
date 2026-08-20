// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package builder_test

import (
	"path/filepath"
	"testing"

	backendgolang "go.thesmos.sh/eidos/backend/golang"
	"go.thesmos.sh/eidos/eidostest/golangtest"
	"go.thesmos.sh/eidos/eidostest/plugintest"
	"go.thesmos.sh/eidos/eidostest/storefixture"
	"go.thesmos.sh/eidos/sdk"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/generator/builder"
	"go.thesmos.sh/testkit/generator/defaults"
	"go.thesmos.sh/testkit/generator/internal/gentest"
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
				BuildStore: func(t *testing.T) *sdk.Store { t.Helper(); return fixture(t, field("Name")) },
			},
			{
				Name: "struct seeded from a companion",
				BuildStore: func(t *testing.T) *sdk.Store {
					t.Helper()
					return withCompanion(t, "Config", "ConfigDefaults", "Config")
				},
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
		primary(t, embeddedPackage()).
			AssertMethod(t, itemBuilder, "WithMeta").
			Signature(t, "(v Meta) *ItemBuilder")
	})

	t.Run("sets a type embedded by pointer", func(t *testing.T) {
		t.Parallel()
		// An embed by pointer records its name on the pointee, so a projection
		// reading the reference's own name drops the field with no diagnostic
		// and the promoted fields become unreachable.
		primary(t, embeddedPackage()).
			AssertMethod(t, itemBuilder, "WithAudit").
			Signature(t, "(v Audit) *ItemBuilder")
	})

	t.Run("allocates for a type embedded by pointer", func(t *testing.T) {
		t.Parallel()
		// The promoted fields are reachable only once the pointer is non-nil, so
		// a setter demanding an address makes every caller allocate first.
		primary(t, embeddedPackage()).
			InMethod(t, itemBuilder, "WithAudit").
			AssertContains(t, "b.v.Audit = &v")
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
		companion(t, plainPackage()).
			InFunc(t, "TestItemBuilderWithName").
			AssertContains(t, `WithName("test-name").Build().Name`)
	})

	t.Run("sets it a second time to a different value", func(t *testing.T) {
		t.Parallel()
		// One value passes whenever the constructor already seeded it, and a
		// companion's return is opaque here — the pair is what covers that.
		companion(t, plainPackage()).
			InFunc(t, "TestItemBuilderWithName").
			AssertContains(t, `WithName("other-name").Build().Name`)
	})

	t.Run("omits the check for a type admitting no pair", func(t *testing.T) {
		t.Parallel()
		// Keeping it would assert, pass, and prove nothing, which reads as
		// coverage the setter does not have.
		companion(t, embeddedPackage()).
			AssertNoSubtest(t, "TestItemBuilderWithMeta", "reaches Meta")
	})

	t.Run("says why the check is absent", func(t *testing.T) {
		t.Parallel()
		// A reader looking for the check finds the reason rather than a gap.
		// Asserted over the file rather than over a scope: the reason is a
		// comment, and a printed function body carries none.
		testkit.Contains(t, text(t, companion(t, embeddedPackage())),
			"No check that the setter reaches Meta", "the absence is explained")
	})

	t.Run("derives a pair once a type parameter resolves to its witness", func(t *testing.T) {
		t.Parallel()
		// The parameter admits no pair at the source, so a projection that did
		// not re-derive after substitution would leave every generic field's
		// setter unchecked.
		companion(t, genericPackage()).
			InFunc(t, "TestBoxBuilderWithValue").
			AssertContains(t, `WithValue("test-value")`)
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
		companion(t, unwritablePackage()).
			InFunc(t, "TestItemBuilderWithEvents").
			AssertContains(t, "ch := make(chan string)").
			AssertContains(t, "WithEvents(ch).Build().Events == ch")
	})

	t.Run("checks a func by arrival", func(t *testing.T) {
		t.Parallel()
		// A func is not comparable, so there is nothing else to assert — but a
		// setter assigning nothing leaves nil, which this catches.
		companion(t, unwritablePackage()).
			InFunc(t, "TestItemBuilderWithCallback").
			AssertContains(t, "Build().Callback != nil")
	})

	t.Run("gives the func literal a body returning its own zero values", func(t *testing.T) {
		t.Parallel()
		// A literal is the only non-nil func available, and its body has to
		// satisfy whatever the field's signature returns.
		companion(t, unwritablePackage()).
			InFunc(t, "TestItemBuilderWithCallback").
			AssertContains(t, "var r0 error").
			AssertContains(t, "return r0")
	})

	t.Run("checks an error by identity", func(t *testing.T) {
		t.Parallel()
		// Two errors carrying the same text are not equal, so the check matches
		// the one it handed over rather than comparing values.
		companion(t, unwritablePackage()).
			InFunc(t, "TestItemBuilderWithErr").
			AssertContains(t, `errors.New("test-Err")`).
			AssertContains(t, "testkit.ErrorIs(")
	})

	t.Run("writes a curated standard-library value", func(t *testing.T) {
		t.Parallel()
		// The resolver never loads the standard library, so a named stdlib type
		// can only be answered by the sampler's curated table. time.Time is in
		// it, which is what turns a dropped check into a written one.
		companion(t, unwritablePackage()).
			InFunc(t, "TestItemBuilderWithAt").
			AssertContains(t, "time.Unix(")
	})

	t.Run("declines a type from a package the run never read", func(t *testing.T) {
		t.Parallel()
		// The floor: nothing about example.com/other.Handle is in the graph and
		// nothing curates it, so no value of it can be written and the check is
		// dropped rather than faked.
		testkit.Contains(t, text(t, companion(t, unwritablePackage())),
			"No check that the setter reaches Handle", "the absence is explained")
	})

	t.Run("checks a directional channel by identity", func(t *testing.T) {
		t.Parallel()
		// The direction is in the stamp rather than in the reference's shape, so
		// a make built from the field's own type renders `make(<-chan T)`, which
		// is not legal Go. The sampler answers the bidirectional form, which
		// assigns to either direction — so this is checkable after all, by the
		// same identity route every other channel takes.
		companion(t, directionalChanPackage()).
			InFunc(t, "TestItemBuilderWithEvents").
			AssertContains(t, "ch := make(chan string)").
			AssertContains(t, "WithEvents(ch).Build().Events == ch")
	})
}

// A map to the empty struct carries its whole meaning in its keys, so a setter
// asking for the value asks the caller for the one thing they cannot vary.
func TestSetField(t *testing.T) {
	t.Parallel()

	t.Run("takes no value parameter on the entry setter", func(t *testing.T) {
		t.Parallel()
		primary(t, setPackage()).
			AssertMethod(t, itemBuilder, "WithTagsEntry").
			Signature(t, "(k string) *ItemBuilder")
	})

	t.Run("adds many keys variadically", func(t *testing.T) {
		t.Parallel()
		// A caller writing map[string]struct{}{"a": {}, "b": {}} at every call
		// site is the reason this shape exists at all.
		primary(t, setPackage()).
			AssertMethod(t, itemBuilder, "WithTagsEntries").
			Signature(t, "(keys ...string) *ItemBuilder")
	})

	t.Run("supplies the value itself", func(t *testing.T) {
		t.Parallel()
		primary(t, setPackage()).
			InMethod(t, itemBuilder, "WithTagsEntry").
			AssertContains(t, "b.v.Tags[k] = struct{}{}")
	})

	t.Run("copies the set on clone", func(t *testing.T) {
		t.Parallel()
		// A set is a map, so a clone sharing it lets one test's keys appear in
		// another's.
		primary(t, setPackage()).
			InMethod(t, itemBuilder, "Clone").
			AssertContains(t, "out.v.Tags = make(map[string]struct{}, len(b.v.Tags))")
	})

	t.Run("leaves a map with a real value type alone", func(t *testing.T) {
		t.Parallel()
		// The narrower reading applies only where the value is the anonymous
		// empty struct; anything else is an ordinary mapping.
		primary(t, setPackage()).
			AssertMethod(t, itemBuilder, "WithMetaEntry").
			Signature(t, "(k string, v string) *ItemBuilder")
	})

	t.Run("checks the set with two distinct keys", func(t *testing.T) {
		t.Parallel()
		companion(t, setPackage()).
			InFunc(t, "TestItemBuilderWithTagsEntries").
			AssertContains(t, `WithTagsEntry("test-tags")`).
			AssertContains(t, `WithTagsEntries("other-tags")`)
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
		companion(t, opaqueSetPackage()).
			InFunc(t, "TestItemBuilderWithKindsEntry").
			AssertContains(t, "var k Kind")
	})

	t.Run("omits the check two keys would be needed for", func(t *testing.T) {
		t.Parallel()
		// Adding one key twice cannot tell an adding setter from a replacing
		// one, so the check would pass against either.
		companion(t, opaqueSetPackage()).
			AssertNoSubtest(t, "TestItemBuilderWithKindsEntries", "keeps keys it was not given")
	})

	t.Run("says why the check is absent", func(t *testing.T) {
		t.Parallel()
		testkit.Contains(t, text(t, companion(t, opaqueSetPackage())),
			"No check that adding keeps what was there", "the absence is explained")
	})
}

// A pointer field distinguishes unset from zero, and the caller who wants to
// say "set" should not have to produce an address to say it.
func TestPointerField(t *testing.T) {
	t.Parallel()

	t.Run("takes the pointee by value", func(t *testing.T) {
		t.Parallel()
		primary(t, pointerPackage()).
			AssertMethod(t, itemBuilder, "WithRetries").
			Signature(t, "(v int) *ItemBuilder")
	})

	t.Run("takes the address itself", func(t *testing.T) {
		t.Parallel()
		primary(t, pointerPackage()).
			InMethod(t, itemBuilder, "WithRetries").
			AssertContains(t, "b.v.Retries = &v")
	})

	t.Run("checks the field through the pointer rather than past it", func(t *testing.T) {
		t.Parallel()
		// A setter that assigned nothing leaves nil, and dereferencing that
		// panics instead of saying which setter failed.
		companion(t, pointerPackage()).
			InFunc(t, "TestItemBuilderWithRetries").
			AssertContains(t, "Build().Retries, &want")
	})

	t.Run("checks that the setter allocated at all", func(t *testing.T) {
		t.Parallel()
		// The one assertion that holds for a pointee admitting no sample pair,
		// which is every pointer to a struct or an interface.
		companion(t, pointerPackage()).
			InFunc(t, "TestItemBuilderWithRetries").
			AssertContains(t, "Build().Retries != nil")
	})

	t.Run("leaves a pointer element inside a slice alone", func(t *testing.T) {
		t.Parallel()
		// The rule applies to a field whose own type is a pointer. An element
		// type is the caller's to supply, so Append keeps taking it as declared.
		primary(t, pointerPackage()).
			AssertMethod(t, itemBuilder, "AppendPeers").
			Signature(t, "(v ...*Item) *ItemBuilder")
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
		primary(t, companionPackage()).
			InFunc(t, "NewItem").
			AssertContains(t, "seed.Seed()")
	})

	t.Run("resolves a bare name against the declaring package", func(t *testing.T) {
		t.Parallel()
		// The form an author writes for a companion beside the struct but named
		// something other than the convention. Nothing qualifies it, so the
		// package that declared the struct is the only package it can mean.
		// Routed out of the declaring package, which is the only arrangement
		// under which the reference has to carry a qualifier — in the
		// same-package case the backend elides it and a companion resolved to
		// no package at all renders identically.
		primary(t, namedCompanionPackage()).
			InFunc(t, "NewItem").
			AssertContains(t, "cfg.Seed()")
	})

	t.Run("reports a qualifier the declaring file does not import", func(t *testing.T) {
		t.Parallel()
		// Resolving it to something plausible emits a reference the file never
		// imports, failing in the consumer's compiler rather than here.
		diags := gentest.Diagnostics(t, builder.New(), unresolvableCompanionPackage())
		testkit.Len(t, diags, 1, "an unresolvable companion is reported once")
	})

	t.Run("declines a convention companion returning something else", func(t *testing.T) {
		t.Parallel()
		// A `ConfigDefaults` returning another type is a different function that
		// happens to collide, and calling it emits a constructor that does not
		// compile.
		diags := gentest.Diagnostics(t, builder.New(), withCompanion(t, "Config", "ConfigDefaults", "Other"))
		testkit.Len(t, diags, 0, "a mismatched companion is passed over, not reported")
	})
}

// Not every declared field earns a setter, and each exclusion has to be
// invisible in the output rather than half-applied.
func TestExcludedFields(t *testing.T) {
	t.Parallel()

	t.Run("drops a field tagged out of the builder", func(t *testing.T) {
		t.Parallel()
		// For a field a test should never set but which cannot be unexported.
		primary(t, skipPackage()).AssertNoMethod(t, itemBuilder, "WithSecret")
	})

	t.Run("keeps the fields beside it", func(t *testing.T) {
		t.Parallel()
		// An opt-out that dropped the whole struct would pass a test asserting
		// only the absence.
		primary(t, skipPackage()).AssertMethod(t, itemBuilder, "WithName")
	})

	t.Run("drops a field whose type the run did not record", func(t *testing.T) {
		t.Parallel()
		// A setter needs a type to declare its parameter and there is nothing
		// to put there. Keeping the field failed the whole run at render, with
		// a message naming a template line rather than the declaration.
		primary(t, skipPackage()).AssertNoMethod(t, itemBuilder, "WithUntyped")
	})

	t.Run("says which field it dropped for want of a type", func(t *testing.T) {
		t.Parallel()
		// A warning rather than silence: the field is in the author's source
		// and its absence from the builder is otherwise unexplained.
		diags := gentest.Diagnostics(t, builder.New(), skipFixture(t))
		testkit.Len(t, diags, 1, "an untyped field is reported once")
		testkit.Contains(t, diags[0].Message, "Item.Untyped", "the diagnostic names the field")
	})

	t.Run("drops an unexported embedded type", func(t *testing.T) {
		t.Parallel()
		// A builder in another package cannot name it, and one in the same
		// package would offer a setter the type's invariants were written to
		// prevent.
		primary(t, embeddedPackage()).AssertNoMethod(t, itemBuilder, "Withaudit")
	})
}

// A struct is opted in by the directive, and a package generally holds more
// declarations than the ones a builder was asked for.
func TestUndirectedStruct(t *testing.T) {
	t.Parallel()

	t.Run("generates nothing for it", func(t *testing.T) {
		t.Parallel()
		// Two files, not four: the annotated struct's pair and nothing for its
		// neighbour. A generator that walked every struct would emit setters
		// for types whose author never asked for them.
		render(t, mixedPackage()).AssertPaths(t,
			"cfg/types"+builder.GoPrimarySuffix, "cfg/types"+builder.GoTestSuffix)
	})

	t.Run("still generates for its annotated neighbour", func(t *testing.T) {
		t.Parallel()
		// The walk has to pass over an undirected struct rather than stop at it.
		primary(t, mixedPackage()).AssertFunc(t, "NewItem")
	})
}

// Swallowing a failed append reads downstream as a struct nobody annotated
// rather than as a fault, and the builder is this plugin's whole output.
func TestGenerateReportsAFailedAppend(t *testing.T) {
	t.Parallel()

	s := fixture(t, field("Name"))
	// Freezing from outside the pipeline stands in for the real cause: an
	// append arriving after Layout has closed the graph.
	s.Emit().Freeze()

	err := builder.New().Generate(&sdk.GeneratorContext{
		Store: s, Reader: sdk.NewStoreReader(s), Diag: sdk.NewSink(),
	})

	testkit.Error(t, err, "a failed append must surface")
	testkit.Contains(t, err.Error(), "Config", "the error must name the declaration")
}

// The substitution is what lets a generic builder's checks be an ordinary
// non-generic test function, and it has to reach inside composites.
func TestGenericSubstitution(t *testing.T) {
	t.Parallel()

	t.Run("rewrites a type parameter inside a slice", func(t *testing.T) {
		t.Parallel()
		companion(t, genericCompositePackage()).
			InFunc(t, "TestBoxBuilderWithItems").
			AssertContains(t, "var v string")
	})

	t.Run("derives a pointer field's pair from its pointee", func(t *testing.T) {
		t.Parallel()
		// The setter takes the pointee, so a pair derived from the field's own
		// type would be a pair of pointers to nothing.
		companion(t, genericCompositePackage()).
			InFunc(t, "TestBoxBuilderWithPtr").
			AssertContains(t, `"test-ptr"`)
	})

	t.Run("derives a set entry's pair from its key", func(t *testing.T) {
		t.Parallel()
		// A set's whole meaning is in its keys, so the key is what the entry
		// setter takes and the only thing a pair can be written for.
		companion(t, genericCompositePackage()).
			InFunc(t, "TestBoxBuilderWithSeenEntry").
			AssertContains(t, `"test-seen"`)
	})

	t.Run("rewrites a type parameter inside a map", func(t *testing.T) {
		t.Parallel()
		// The key and the value are separate references, and a rewrite that
		// reached only one would leave the other naming a parameter no longer
		// in scope. Asserted on the declared locals rather than on the rendered
		// `map[string]string`, which a comment can split when the body is
		// re-printed.
		companion(t, genericCompositePackage()).
			InFunc(t, "TestBoxBuilderWithBy").
			AssertContains(t, "var k string").
			AssertContains(t, "var v string")
	})
}

// The diagnostics are the one behaviour the corpus cannot show: a fixture that
// provokes one would fail the run that generates every other fixture.
func TestDiagnostics(t *testing.T) {
	t.Parallel()

	t.Run("reports a struct with nothing to set", func(t *testing.T) {
		t.Parallel()
		// A builder with no setters configures nothing, and emitting the shell
		// would hide a declaration that cannot do what it says.
		diags := gentest.Diagnostics(t, builder.New(), fixture(t, field("secret")))
		testkit.Len(t, diags, 1, "a builder with no fields is reported")
	})

	t.Run("rejects a tag value that is not the opt-out", func(t *testing.T) {
		t.Parallel()
		// Silently keeping the setter would leave the author believing a field
		// they meant to exclude is excluded.
		diags := gentest.Diagnostics(t, builder.New(), fixture(t, tagged("Name", `builder:"skip"`)))
		testkit.Len(t, diags, 1, "a mistyped opt-out is reported")
	})
}

// Rendering is where a generator actually fails. Every assertion driven off the
// emit graph passes against a template that renders code which does not compile
// — an undeclared local, a setter whose receiver disagrees with its return.
// Those surface only once the backend runs, so the templates are driven
// end-to-end here rather than trusted.
func TestRender(t *testing.T) {
	t.Parallel()

	t.Run("renders both outputs and nothing else", func(t *testing.T) {
		t.Parallel()
		render(t, plainPackage()).
			AssertPaths(t, sourceDir+"/"+primaryFile, sourceDir+"/"+companionFile)
	})

	t.Run("renders without a diagnostic", func(t *testing.T) {
		t.Parallel()
		// Driven through the builder rather than through Render: the latter
		// stops the test on an error diagnostic, which would leave a warning
		// invisible.
		run := golangtest.Driver(t, backendgolang.New(), plainPackage(), builder.New()).
			Build().Run("./...")
		testkit.Len(t, run.Diagnostics().Diagnostics(), 0,
			"a clean fixture renders without diagnostics")
	})

	// Deliberately one subtest and deliberately serial. A [golangtest.Generated]
	// caches its assembled module under the [testing.TB] that first built it,
	// so splitting these would leave the later ones pointed at a TempDir the
	// earlier one had removed.
	t.Run("the toolchain accepts what it emits", func(t *testing.T) {
		t.Parallel()
		// The assertion every token check above stands on: a setter whose
		// receiver disagrees with its return, or a Clone naming a field that
		// moved, satisfies every substring claim and still does not build. The
		// hand-written package is projected from the fixture that drove the
		// run, so the two cannot drift apart.
		//
		// Vetted and run as well as compiled, because half of what this plugin
		// emits is itself a test file. A generated check that compiles and
		// fails, or that vet reports, is a defect delivered into a consumer's
		// repository against a file nobody there wrote — and compiling alone
		// cannot tell the three apart.
		gen := render(t, plainPackage()).
			WithSource(golangtest.GoFile(plainFixture().GoSource())).
			WithRequire(builder.Module, filepath.Join("..", ".."))

		gen.AssertCompiles(t)
		gen.AssertVets(t)
		gen.AssertTestsPass(t)
	})

	t.Run("emits gofmt-clean source", func(t *testing.T) {
		t.Parallel()
		// A generator whose output the toolchain reformats produces a diff on
		// every run of every consuming repository.
		primary(t, plainPackage()).AssertFormatted(t)
		companion(t, plainPackage()).AssertFormatted(t)
	})

	t.Run("emits a setter shaped by each field's type", func(t *testing.T) {
		t.Parallel()
		f := primary(t, plainPackage())
		for _, want := range []struct{ method, signature string }{
			{"WithName", "(v string) *ItemBuilder"},
			{"WithTags", "(v ...string) *ItemBuilder"},
			{"AppendTags", "(v ...string) *ItemBuilder"},
			{"WithBodyString", "(s string) *ItemBuilder"},
			{"WithMetaEntry", "(k string, v string) *ItemBuilder"},
			{"WithMetaEntries", "(entries map[string]string) *ItemBuilder"},
			{"Clone", "() *ItemBuilder"},
		} {
			f.AssertMethod(t, itemBuilder, want.method).Signature(t, want.signature)
		}
	})

	t.Run("keeps an unexported field out of the builder", func(t *testing.T) {
		t.Parallel()
		// A setter in another package could not name it, and one in the same
		// package would offer a way past the invariants it exists to protect.
		primary(t, plainPackage()).AssertNoMethod(t, itemBuilder, "Withhidden")
	})

	t.Run("copies every field that owns storage", func(t *testing.T) {
		t.Parallel()
		// A clone sharing a slice or map lets one test's setup appear in
		// another's, which surfaces as a failure for something it never did.
		primary(t, plainPackage()).
			InMethod(t, itemBuilder, "Clone").
			AssertContains(t, "out.v.Tags = append([]string(nil), b.v.Tags...)").
			AssertContains(t, "out.v.Meta = make(map[string]string, len(b.v.Meta))")
	})

	t.Run("seeds the constructor from a declared default", func(t *testing.T) {
		t.Parallel()
		primary(t, seededPackage()).
			InFunc(t, "NewItem").
			AssertContains(t, `Name: "seed"`)
	})

	t.Run("qualifies a default naming a symbol elsewhere", func(t *testing.T) {
		t.Parallel()
		// The stamp carries the symbol and its import path apart, because a
		// rendered file has to register the import and only a reference can ask
		// for one. Concatenating the two into the stamp would emit a name the
		// file never imports.
		primary(t, seededPackage()).
			InFunc(t, "NewItem").
			AssertContains(t, "Region: seed.Region")
	})

	t.Run("instantiates a generic builder's checks at concrete types", func(t *testing.T) {
		t.Parallel()
		// A Go test function cannot take type parameters, so a check naming the
		// parameter in a field position would not compile.
		companion(t, genericPackage()).
			InFunc(t, "TestNewBox").
			AssertContains(t, "NewBox[string]()")
	})

	t.Run("declines to check a builder it cannot instantiate", func(t *testing.T) {
		t.Parallel()
		// The absence is stated rather than left as an empty file, which would
		// read as a generator that failed. It is a comment, so the whole file
		// is what carries it.
		testkit.Contains(t, text(t, companion(t, boundedPackage())),
			"Skipped:", "an uninstantiable builder says so")
	})
}

// The goldens are the readable record of what this generator produces. A diff
// on them is the review surface for any template change — the assertions above
// say a construct is present, and only the golden says what the whole file
// reads like.
//
// Regenerate by deleting the file, which keeps the change visible in review.
func TestRenderMatchesGolden(t *testing.T) {
	t.Parallel()

	t.Run("the builder", func(t *testing.T) {
		t.Parallel()
		primary(t, plainPackage()).AssertGolden(t, "testdata/golden/"+primaryFile)
	})

	t.Run("the checks", func(t *testing.T) {
		t.Parallel()
		companion(t, plainPackage()).AssertGolden(t, "testdata/golden/"+companionFile)
	})
}

// The rendered filenames, composed by Layout from the source basename and the
// adapter's suffixes. Every fixture declares its struct in `cfg/types.go`, so
// the run resolves both targets into that directory.
const (
	sourceDir     = "cfg"
	primaryFile   = "types" + builder.GoPrimarySuffix
	companionFile = "types" + builder.GoTestSuffix
)

// itemBuilder is the receiver every fixture's setters hang off.
const itemBuilder = "ItemBuilder"

// render drives the plugin and the Go backend over pkg and adopts the files the
// run produced, so routing and rendering both participate.
func render(t *testing.T, pkg *sdk.Package) *golangtest.Generated {
	t.Helper()
	return golangtest.Render(t, backendgolang.New(), pkg, builder.New())
}

// primary parses the builder this run wrote.
//
// Addressed by the adapter's own suffix rather than by path: Layout composes
// the rest of the name from a source basename and routes it into the source's
// directory, neither of which this plugin declares.
func primary(t *testing.T, pkg *sdk.Package) *golangtest.Source {
	t.Helper()
	return render(t, pkg).Suffixed(t, builder.GoPrimarySuffix)
}

// companion parses the checks this run wrote.
func companion(t *testing.T, pkg *sdk.Package) *golangtest.Source {
	t.Helper()
	return render(t, pkg).Suffixed(t, builder.GoTestSuffix)
}

// text returns a rendered file as a string, for the claims a scoped assertion
// cannot make: a printed function body carries no comments, and several of this
// generator's decisions are recorded as one.
func text(t *testing.T, src *golangtest.Source) string {
	t.Helper()
	return string(src.Bytes())
}

// plainPackage carries one field of every shape that changes a setter.
func plainPackage() *sdk.Package { return plainFixture().PackageNode() }

// plainFixture is [plainPackage] as the builder that declared it, so the
// hand-written package the generated output references can be projected from
// the same declaration that drove the run. Written by hand the two are bound
// only by review, and a renamed field surfaces as a compile error inside a
// throwaway module naming code nobody wrote.
func plainFixture() *storefixture.Builder {
	return storefixture.New().
		Package("cfg", "example.com/cfg").
		Struct("Item", func(b *storefixture.StructBuilder) {
			// Layout composes the filename from the source basename, so the
			// fixture needs a position for the rendered name to be anything
			// other than a bare suffix.
			b.Pos(sdk.At("cfg/types.go", 1, 1))
			b.Directive(storefixture.Directive("builder"))
			b.Field("Name", storefixture.Named("string"), nil)
			b.Field("Tags", storefixture.Slice(storefixture.Named("string")), nil)
			b.Field("Body", storefixture.Slice(storefixture.Named("byte")), nil)
			b.Field("Meta", storefixture.Map(storefixture.Named("string"), storefixture.Named("string")), nil)
			b.Field("hidden", storefixture.Named("string"), nil)
		})
}

// embeddedPackage carries a struct embedding another type both ways.
func embeddedPackage() *sdk.Package {
	return storefixture.New().
		Package("cfg", "example.com/cfg").
		Struct("Item", func(b *storefixture.StructBuilder) {
			b.Pos(sdk.At("cfg/types.go", 1, 1))
			b.Directive(storefixture.Directive("builder"))
			b.Embed(storefixture.Named("Meta"))
			b.Embed(storefixture.Pointer(storefixture.Named("Audit")))
			// Unexported, so a builder in another package cannot name it and
			// one in the same package would offer a setter the type's own
			// invariants were written to prevent.
			b.Embed(storefixture.Named("audit"))
			b.Field("Name", storefixture.Named("string"), nil)
		}).
		PackageNode()
}

// unwritablePackage carries the field types no literal can be written for.
func unwritablePackage() *sdk.Package {
	return storefixture.New().
		Package("cfg", "example.com/cfg").
		Struct("Item", func(b *storefixture.StructBuilder) {
			b.Pos(sdk.At("cfg/types.go", 1, 1))
			b.Directive(storefixture.Directive("builder"))
			b.Field("Events", storefixture.Chan(storefixture.Named("string")), nil)
			b.Field("Callback", storefixture.Func(
				[]*sdk.TypeRef{storefixture.Named("int")},
				[]*sdk.TypeRef{storefixture.Named("error")},
			), nil)
			b.Field("Err", storefixture.Named("error"), nil)
			b.Field("At", storefixture.PkgNamed("time", "Time"), nil)
			b.Field("Handle", storefixture.PkgNamed("example.com/other", "Handle"), nil)
		}).
		PackageNode()
}

// directionalChanPackage carries a receive-only channel, which takes a setter
// like any other but no check that has to construct one.
func directionalChanPackage() *sdk.Package {
	return storefixture.New().
		Package("cfg", "example.com/cfg").
		Struct("Item", func(b *storefixture.StructBuilder) {
			b.Pos(sdk.At("cfg/types.go", 1, 1))
			b.Directive(storefixture.Directive("builder"))
			b.Field("Events", storefixture.RecvChan(storefixture.Named("string")), nil)
		}).
		PackageNode()
}

// setPackage carries a set beside an ordinary map, which take different
// setters and are told apart by the map's value type rather than by its name.
func setPackage() *sdk.Package {
	return storefixture.New().
		Package("cfg", "example.com/cfg").
		Struct("Item", func(b *storefixture.StructBuilder) {
			b.Pos(sdk.At("cfg/types.go", 1, 1))
			b.Directive(storefixture.Directive("builder"))
			b.Field("Tags", storefixture.Map(storefixture.Named("string"), emptyStruct()), nil)
			b.Field("Meta", storefixture.Map(storefixture.Named("string"), storefixture.Named("string")), nil)
		}).
		PackageNode()
}

// opaqueSetPackage keys its set by a type no sample pair can be written for.
func opaqueSetPackage() *sdk.Package {
	return storefixture.New().
		Package("cfg", "example.com/cfg").
		Struct("Item", func(b *storefixture.StructBuilder) {
			b.Pos(sdk.At("cfg/types.go", 1, 1))
			b.Directive(storefixture.Directive("builder"))
			b.Field("Kinds", storefixture.Map(storefixture.Named("Kind"), emptyStruct()), nil)
		}).
		PackageNode()
}

// emptyStruct builds the anonymous `struct{}` the frontend records for a set's
// value type.
func emptyStruct() *sdk.TypeRef { return storefixture.AnonStruct(nil, nil) }

// pointerPackage carries a pointer field and a slice of pointers, which take
// different setters: the rule applies to a field that is itself a pointer.
func pointerPackage() *sdk.Package {
	return storefixture.New().
		Package("cfg", "example.com/cfg").
		Struct("Item", func(b *storefixture.StructBuilder) {
			b.Pos(sdk.At("cfg/types.go", 1, 1))
			b.Directive(storefixture.Directive("builder"))
			b.Field("Retries", storefixture.Pointer(storefixture.Named("int")), nil)
			b.Field("Peers", storefixture.Slice(storefixture.Pointer(storefixture.Named("Item"))), nil)
		}).
		PackageNode()
}

// companionPackage names its companion by full import path.
func companionPackage() *sdk.Package {
	return storefixture.New().
		Package("cfg", "example.com/cfg").
		Struct("Item", func(b *storefixture.StructBuilder) {
			b.Pos(sdk.At("cfg/types.go", 1, 1))
			b.Directive(storefixture.Directive("builder",
				storefixture.KV("defaults", "example.com/seed.Seed")))
			b.Field("Name", storefixture.Named("string"), nil)
		}).
		PackageNode()
}

// namedCompanionPackage names its companion by a bare identifier, which is the
// form for one beside the struct under a name the convention would not derive.
func namedCompanionPackage() *sdk.Package {
	return storefixture.New().
		Package("cfg", "example.com/cfg").
		Struct("Item", func(b *storefixture.StructBuilder) {
			b.Pos(sdk.At("cfg/types.go", 1, 1))
			b.Directive(storefixture.Directive("out",
				storefixture.Arg("cfgtest/"),
				storefixture.KV("pkg", "cfgtest"),
			))
			b.Directive(storefixture.Directive("builder",
				storefixture.KV("defaults", "Seed")))
			b.Field("Name", storefixture.Named("string"), nil)
		}).
		PackageNode()
}

// unresolvableCompanionPackage qualifies its companion against a package the
// declaring file does not import.
func unresolvableCompanionPackage() *sdk.Store {
	return storefixture.New().
		Package("cfg", "example.com/cfg").
		Struct("Item", func(b *storefixture.StructBuilder) {
			b.Pos(sdk.At("cfg/types.go", 1, 1))
			b.Directive(storefixture.Directive("builder",
				storefixture.KV("defaults", "nowhere.Seed")))
			b.Field("Name", storefixture.Named("string"), nil)
		}).
		Build()
}

// mixedPackage holds an annotated struct beside one carrying no directive,
// which is what an ordinary package looks like.
func mixedPackage() *sdk.Package {
	return storefixture.New().
		Package("cfg", "example.com/cfg").
		Struct("Item", func(b *storefixture.StructBuilder) {
			b.Pos(sdk.At("cfg/types.go", 1, 1))
			b.Directive(storefixture.Directive("builder"))
			b.Field("Name", storefixture.Named("string"), nil)
		}).
		Struct("Internal", func(b *storefixture.StructBuilder) {
			b.Pos(sdk.At("cfg/types.go", 1, 1))
			b.Field("Name", storefixture.Named("string"), nil)
		}).
		PackageNode()
}

// seededPackage carries one field declaring a plain default and one declaring
// a default that names a symbol in another package.
//
// The second travels as two stamps, because a rendered file has to register the
// import and only a reference can carry one.
func seededPackage() *sdk.Package {
	pkg := storefixture.New().
		Package("cfg", "example.com/cfg").
		Struct("Item", func(b *storefixture.StructBuilder) {
			b.Pos(sdk.At("cfg/types.go", 1, 1))
			b.Directive(storefixture.Directive("builder"))
			b.Field("Name", storefixture.Named("string"), nil)
			b.Field("Region", storefixture.Named("string"), nil)
		}).
		PackageNode()
	for _, s := range pkg.Structs {
		defaults.MetaDefault.Set(s.Fields[0].EnsureMeta(), `"seed"`, "test")
		defaults.MetaDefault.Set(s.Fields[1].EnsureMeta(), "Region", "test")
		defaults.MetaDefaultPkg.Set(s.Fields[1].EnsureMeta(), "example.com/seed", "test")
	}
	return pkg
}

// genericPackage carries a struct whose constraints admit a witness.
func genericPackage() *sdk.Package {
	return storefixture.New().
		Package("cfg", "example.com/cfg").
		Struct("Box", func(b *storefixture.StructBuilder) {
			b.Pos(sdk.At("cfg/types.go", 1, 1))
			b.Directive(storefixture.Directive("builder"))
			b.TypeParam("T", storefixture.Bound("any", storefixture.Named("any")))
			b.Field("Value", storefixture.Named("T"), nil)
		}).
		PackageNode()
}

// genericCompositePackage parameterises a slice and a map.
func genericCompositePackage() *sdk.Package {
	return storefixture.New().
		Package("cfg", "example.com/cfg").
		Struct("Box", func(b *storefixture.StructBuilder) {
			b.Pos(sdk.At("cfg/types.go", 1, 1))
			b.Directive(storefixture.Directive("builder"))
			b.TypeParam("T", storefixture.Bound("any", storefixture.Named("any")))
			b.Field("Items", storefixture.Slice(storefixture.Named("T")), nil)
			b.Field("By", storefixture.Map(storefixture.Named("T"), storefixture.Named("T")), nil)
			// A pointer and a set take their sample from what the setter
			// actually receives — the pointee and the key — so a substitution
			// reading the field's own type derives a pair for neither.
			b.Field("Ptr", storefixture.Pointer(storefixture.Named("T")), nil)
			b.Field("Seen", storefixture.Map(storefixture.Named("T"), emptyStruct()), nil)
		}).
		PackageNode()
}

// boundedPackage carries a struct bounded by a constraint no witness satisfies.
func boundedPackage() *sdk.Package {
	return storefixture.New().
		Package("cfg", "example.com/cfg").
		Struct("Ranked", func(b *storefixture.StructBuilder) {
			b.Pos(sdk.At("cfg/types.go", 1, 1))
			b.Directive(storefixture.Directive("builder"))
			b.TypeParam("K", storefixture.Bound("Ordered", storefixture.Named("Ordered")))
			b.Field("Key", storefixture.Named("K"), nil)
		}).
		PackageNode()
}

// skipPackage carries a field opted out through the struct tag beside one that
// is not, and a field whose type the frontend could not record.
func skipPackage() *sdk.Package {
	return storefixture.New().
		Package("cfg", "example.com/cfg").
		Struct("Item", func(b *storefixture.StructBuilder) {
			b.Pos(sdk.At("cfg/types.go", 1, 1))
			b.Directive(storefixture.Directive("builder"))
			b.Field("Name", storefixture.Named("string"), nil)
			b.Field("Secret", storefixture.Named("string"), func(f *storefixture.FieldBuilder) {
				f.Tag(`builder:"-"`)
			})
			b.Field("Untyped", nil, nil)
		}).
		PackageNode()
}

// skipFixture is [skipPackage] as a store, for the assertions whose subject is
// what the run reported rather than what it wrote.
func skipFixture(t *testing.T) *sdk.Store {
	t.Helper()
	return storefixture.New().
		Package("cfg", "example.com/cfg").
		Struct("Item", func(b *storefixture.StructBuilder) {
			b.Directive(storefixture.Directive("builder"))
			b.Field("Name", storefixture.Named("string"), nil)
			b.Field("Untyped", nil, nil)
		}).
		Build()
}

// field returns a builder option declaring one string field.
func field(name string) func(*storefixture.StructBuilder) {
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
func fixture(t *testing.T, opts ...func(*storefixture.StructBuilder)) *sdk.Store {
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
func withCompanion(t *testing.T, name, companionName, returns string) *sdk.Store {
	t.Helper()
	return storefixture.New().
		Package("cfg", "example.com/cfg").
		Struct(name, func(b *storefixture.StructBuilder) {
			b.Directive(storefixture.Directive("builder"))
			b.Field("Host", storefixture.Named("string"), nil)
		}).
		Function(companionName, func(f *storefixture.FunctionBuilder) {
			f.Return(storefixture.Named(returns))
		}).
		Build()
}

// diagnostics drives the plugin over s and returns what it reported.
//
// Generate is called directly rather than through a pipeline: every fixture
// here provokes an error, and a pipeline harness that adopts the output would
// stop the test before the diagnostic could be read.
