// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package suite_test

import (
	"path/filepath"
	"testing"

	backendgolang "go.thesmos.sh/eidos/backend/golang"
	"go.thesmos.sh/eidos/eidostest/golangtest"
	"go.thesmos.sh/eidos/eidostest/plugintest"
	"go.thesmos.sh/eidos/eidostest/storefixture"
	"go.thesmos.sh/eidos/sdk"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/generator/internal/gentest"
	"go.thesmos.sh/testkit/generator/stub"
	"go.thesmos.sh/testkit/generator/suite"
)

// The framework conformance suites. The framework checks pin the static
// contract — stable Name, role implementation, deterministic Outputs,
// well-formed multi-output shape — and the generator suite pins
// determinism, a frozen source store, and diagnostic discipline.
func TestConformance(t *testing.T) {
	t.Parallel()

	t.Run("satisfies the framework contracts", func(t *testing.T) {
		t.Parallel()
		plugintest.RunSuite(t, suite.New())
	})

	t.Run("satisfies the generator contracts", func(t *testing.T) {
		t.Parallel()
		plugintest.RunGeneratorSuite(t, suite.New(), []plugintest.GeneratorFixture{
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
}

// The version composes into the plugin's cache key, so a stale one serves
// output produced by a plugin that has since changed. It also renders into
// every generated file's header.
func TestVersion(t *testing.T) {
	t.Parallel()

	t.Run("declares a non-empty version", func(t *testing.T) {
		t.Parallel()
		testkit.Assert(t, suite.New().Version()).
			IsNotEmpty("a plugin without a version cannot invalidate its cache")
	})

	t.Run("reports the declared constant", func(t *testing.T) {
		t.Parallel()
		testkit.Equal(t, suite.New().Version(), suite.Version, "the method reports the constant")
	})
}

// The end-to-end assertion: render the harness and its proofs companion
// over one interface, build the pair in a throwaway module, and run it.
//
// Nothing weaker would do. The plugin's own unit tests pin the projection
// — what a deriver licensed, what the index declares — and every one of
// them passes against a file that does not compile. The corpus catches
// that, but the corpus is a separate module and a separate command; a
// generator whose only compile gate is somebody remembering to regenerate
// is a generator with no gate.
//
// The double is rendered alongside because the proofs are written against
// it. That is not the test reaching for a second plugin: it is the shape
// of what this one emits, which names a constructor and a per-method
// option another generator owns, and a test that stubbed them out would
// stop noticing the day one of those names moved.
func TestGeneratedHarnessBuildsAndItsProofsRun(t *testing.T) {
	t.Parallel()

	gen := generatedModule(t)

	t.Run("emits the harness and its companion", func(t *testing.T) {
		t.Parallel()
		// Four files from two plugins, in the package the fixture routes
		// to. The pair matters more than either half: a harness with no
		// companion is a set of claims nothing has driven, and a
		// companion with no harness is a file that does not compile.
		gen.AssertPaths(t,
			filepath.Join("storepkg", "storetest", "store"+suite.GoPrimarySuffix),
			filepath.Join("storepkg", "storetest", "store"+suite.GoTestSuffix),
			filepath.Join("storepkg", "storetest", "store"+stub.GoPrimarySuffix),
			filepath.Join("storepkg", "storetest", "store"+stub.GoTestSuffix),
		)
	})

	t.Run("compiles", func(t *testing.T) {
		t.Parallel()
		gen.AssertCompiles(t)
	})

	t.Run("vets", func(t *testing.T) {
		t.Parallel()
		gen.AssertVets(t)
	})
}

// The generated companion RUNS: its proofs drive every claim to a red,
// and its self-check holds the package to what it says about itself.
//
// The one assertion that measures the claims rather than the code.
// prove.All fails a check stamped Proven with no defect, a defect naming
// no check, and a defect the check tolerated — so a green here means the
// emitted claims and the emitted evidence agree, which is the property
// the whole falsifiability tier exists for.
//
// Serial, and its own test rather than a subtest of the one above,
// because it sets an environment variable and [testing.T.Setenv] refuses
// that under any parallel parent. What it sets is the manifest's seed
// mode: a fresh module has no checks.lock, which is exactly the state a
// consumer is in before their first regeneration, and seeding it here
// keeps this about the proofs rather than about a file the harness never
// wrote.
func TestGeneratedCompanionRuns(t *testing.T) {
	t.Setenv("TESTKIT_LOCK_WRITE", "1")
	generatedModule(t).AssertTestsPass(t)
}

// A generic subject gets a companion carrying a note instead of proofs.
//
// A Go test function takes no type parameters, so there is nothing to
// instantiate a planted defect at. The file is emitted anyway: a
// companion that silently vanished would read as a generator that failed,
// and the reader deserves the fact that stopped it.
func TestGenericSubjectGetsAnExplainedCompanion(t *testing.T) {
	t.Parallel()

	gen := golangtest.Render(t, backendgolang.New(), genericBuilder().PackageNode(),
		suite.New(), stub.New()).
		WithSource(golangtest.GoFile(genericBuilder().GoSource())).
		WithRequire(suite.Module, filepath.Join("..", "..")).
		WithRequire(suite.EngineModule, filepath.Join("..", "..", "engine"))

	gen.Suffixed(t, suite.GoTestSuffix).
		AssertContains(t, "is generic").
		AssertNotContains(t, "prove.Defects")
}

// fixturePkg is the import path the fixture's source package resolves at.
const fixturePkg = "example.com/storepkg"

// storeFixture is one `//testkit:suite` interface with a context-taking
// writer, a reader and a void method — enough shape for the smoke family,
// all three context arms, and a method that takes nothing.
func storeFixture(t *testing.T) *sdk.Store {
	t.Helper()
	return fixtureBuilder().Build()
}

func fixtureBuilder() *storefixture.Builder {
	return storefixture.New().
		Package("storepkg", fixturePkg).
		Interface("Store", func(i *storefixture.InterfaceBuilder) {
			// Layout composes the output filename from the source
			// basename, so the fixture needs a position for the rendered
			// names to be anything other than a bare suffix.
			i.Pos(gentest.AtFile("storepkg/store.go"))
			// Routed into a package of its own, which is where a real run
			// puts it and which the harness needs to be legal: it declares
			// `type Store = storepkg.Store`, and a file emitted beside the
			// source would redeclare the name it aliases.
			i.Directive(storefixture.Directive("out",
				storefixture.Arg("storetest/"), storefixture.KV("pkg", "storetest")))
			i.Directive(storefixture.Directive("suite"))
			i.Directive(storefixture.Directive("stub"))
			i.Method("Put", func(m *storefixture.MethodBuilder) {
				gentest.Ctx(m)
				m.Param("id", storefixture.Named("string"))
				gentest.Err(m)
			})
			i.Method("Get", func(m *storefixture.MethodBuilder) {
				gentest.Ctx(m)
				m.Param("id", storefixture.Named("string"))
				m.Return(storefixture.Named("string"))
				gentest.Err(m)
			})
			i.Method("Close", nil)
		})
}

func genericBuilder() *storefixture.Builder {
	return storefixture.New().
		Package("boxpkg", "example.com/boxpkg").
		Interface("Box", func(i *storefixture.InterfaceBuilder) {
			i.Pos(gentest.AtFile("boxpkg/box.go"))
			i.Directive(storefixture.Directive("suite"))
			i.TypeParam("V", storefixture.Bound("any", storefixture.Named("any")))
			i.Method("Put", func(m *storefixture.MethodBuilder) {
				gentest.Ctx(m)
				m.Param("v", storefixture.Named("V"))
				gentest.Err(m)
			})
		})
}

// generatedModule renders the harness, its proofs and the double they are
// written against into one throwaway module.
//
// WithRequire wires this checkout in rather than resolving a published
// version: the sandbox runs with the proxy off, and the assertion is only
// honest if it builds against the runtime in this tree — the one whose
// primitives the proofs quote by constant.
func generatedModule(t *testing.T) *golangtest.Generated {
	t.Helper()
	b := fixtureBuilder()
	return golangtest.Render(t, backendgolang.New(), b.PackageNode(), suite.New(), stub.New()).
		WithSource(golangtest.GoFile(b.GoSource())).
		WithRequire(suite.Module, filepath.Join("..", "..")).
		WithRequire(suite.EngineModule, filepath.Join("..", "..", "engine"))
}
