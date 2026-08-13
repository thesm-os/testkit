// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package sentinel_test

import (
	"path/filepath"
	"strings"
	"testing"

	backendgolang "go.thesmos.sh/eidos/backend/golang"
	"go.thesmos.sh/eidos/eidostest/golangtest"
	"go.thesmos.sh/eidos/eidostest/pipelinetest"
	"go.thesmos.sh/eidos/eidostest/plugintest"
	"go.thesmos.sh/eidos/eidostest/storefixture"
	"go.thesmos.sh/eidos/lang/golang"
	"go.thesmos.sh/eidos/sdk"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/generator/sentinel"
)

// suite is the umbrella check's identifier, composed the way the template
// composes it. Spelled through [golang.TestFuncName] rather than written out,
// because a test asserting on a name the generator derives should derive it
// the same way — otherwise the two drift and the assertion pins the literal.
var suite = golang.TestFuncName(fixturePkgName, "Sentinels") //nolint:gochecknoglobals // derived constant.

// The fixture package's identity. Layout composes the output filename from a
// declaration's source basename, so the position matters as much as the name.
const (
	fixturePkgName = "cfg"
	fixturePkgPath = "example.com/cfg"
	fixtureFile    = "cfg/errors.go"
	renderedFile   = "errors" + sentinel.GoSuffix
)

// The framework conformance suites pin the static contract — a stable name,
// deterministic outputs, templates that parse, two directive schemas that do
// not collide — none of which a fixture-driven test would notice.
func TestConformance(t *testing.T) {
	t.Parallel()

	t.Run("satisfies the framework contracts", func(t *testing.T) {
		t.Parallel()
		plugintest.RunSuite(t, sentinel.New())
	})

	t.Run("satisfies the generator contracts", func(t *testing.T) {
		t.Parallel()
		plugintest.RunGeneratorSuite(t, sentinel.New(), []plugintest.GeneratorFixture{
			{
				Name: "annotated package",
				BuildStore: func(t *testing.T) *sdk.Store {
					t.Helper()
					return bare("").Build()
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

// The prefix key carries two behaviours behind one name, and the difference
// between them is a check that exists and a check that does not.
func TestPrefix(t *testing.T) {
	t.Parallel()

	t.Run("derives the prefix from the package name", func(t *testing.T) {
		t.Parallel()
		rendered(t, bare("")).InFunc(t, suite).AssertContains(t, `"cfg: "`)
	})

	t.Run("takes an override over the derived name", func(t *testing.T) {
		t.Parallel()
		// For a package whose errors are named for the subsystem rather than
		// the directory it happens to live in.
		rendered(t, bare("store")).InFunc(t, suite).AssertContains(t, `"store: "`)
	})

	t.Run("suppresses the check on an empty prefix", func(t *testing.T) {
		t.Parallel()
		// `prefix=` and `prefix=off` say the same thing, and an author who
		// writes the first should not get an assertion they meant to remove.
		rendered(t, bare(emptyPrefix)).
			AssertNoSubtest(t, suite, "every sentinel carries the package prefix")
	})

	t.Run("suppresses the check on prefix=off", func(t *testing.T) {
		t.Parallel()
		rendered(t, bare(sentinel.PrefixOff)).
			AssertNoSubtest(t, suite, "every sentinel carries the package prefix")
	})

	t.Run("says why the suppressed check is absent", func(t *testing.T) {
		t.Parallel()
		// A reader looking for the check finds the reason rather than a gap. A
		// generator that quietly dropped a failing one would be worse than one
		// that never had it.
		//
		// Asserted over the whole file rather than through a scope: the reason
		// is a comment, and a rendered function body carries none.
		rendered(t, bare(sentinel.PrefixOff)).AssertContains(t, "declares prefix=off")
	})
}

// Every sentinel set earns the same checks, and each has to be able to
// fail — which is what the fixture-driven corpus proves and this pins.
func TestSentinelChecks(t *testing.T) {
	t.Parallel()

	t.Run("ignores a variable outside the naming convention", func(t *testing.T) {
		t.Parallel()
		// The Err prefix is what opts a variable in. One named otherwise is
		// not a sentinel, and treating it as one would assert a message
		// contract over something that never claimed to have it.
		b := bare("")
		b.Variable("Timeout", pos)
		rendered(t, b).InFunc(t, suite).AssertNotContains(t, "Timeout")
	})

	t.Run("reads the prefix eidos composes a sentinel name with", func(t *testing.T) {
		t.Parallel()
		// The detector and eidos's own composer are one convention. A
		// divergence would leave this plugin blind to every sentinel an eidos
		// generator emitted, and neither package would say so — the corpus
		// would simply report a package with nothing to check.
		testkit.Equal(t, golang.SentinelName("gone"), sentinel.ErrPrefix+"Gone",
			"the prefix read here is the one eidos writes")
	})

	t.Run("ignores an unexported error variable", func(t *testing.T) {
		t.Parallel()
		// A consumer cannot name it, so a check in their external test package
		// would not compile.
		b := bare("")
		b.Variable("errInternal", pos)
		rendered(t, b).InFunc(t, suite).AssertNotContains(t, "errInternal")
	})

	t.Run("lists every sentinel it found", func(t *testing.T) {
		t.Parallel()
		// A sentinel named outside the convention is not found, so the list is
		// how an absence becomes visible.
		body := rendered(t, bare("")).InFunc(t, suite)
		for _, name := range []string{"ErrEmpty", "ErrFull", "ErrInvalid"} {
			body.AssertContains(t, name)
		}
	})

	t.Run("checks every property a sentinel set owes", func(t *testing.T) {
		t.Parallel()
		f := rendered(t, bare(""))
		for _, want := range []string{
			"every sentinel is non-nil",
			"every sentinel has a message",
			"no two sentinels share a message",
			"no sentinel's message is a prefix of another's",
			"no sentinel matches another",
		} {
			f.AssertSubtest(t, suite, want)
		}
	})

	t.Run("runs every check it emits in parallel", func(t *testing.T) {
		t.Parallel()
		// A generated suite that serialises costs every consumer the same
		// wall-clock on every run, and nothing in the emitted text says so.
		rendered(t, bare("")).AssertParallel(t, suite)
	})

	t.Run("omits the checks that cannot fail", func(t *testing.T) {
		t.Parallel()
		// errors.Is compares identity before consulting anything a type
		// declares, so a sentinel survives %w and errors.Join no matter what
		// it is. The assertion would be about the standard library.
		f := rendered(t, bare(""))
		f.AssertNoSubtest(t, suite, "survives being wrapped")
		f.AssertNoSubtest(t, suite, "survives being joined")
	})

	t.Run("emits nothing about error types when there are none", func(t *testing.T) {
		t.Parallel()
		// The floor. A corpus holding only the rich case cannot tell "the
		// optional checks were correctly omitted" from "silently dropped".
		rendered(t, bare("")).AssertTestFuncs(t, suite)
	})
}

// An optional method earns a check, and a type without it must get no check
// rather than one that cannot fail.
func TestErrorTypes(t *testing.T) {
	t.Parallel()

	t.Run("holds an error type to the same prefix as a sentinel", func(t *testing.T) {
		t.Parallel()
		// A custom error reaches the same logs and is read the same way, so
		// checking only the Err* vars leaves half a package unexamined.
		rendered(t, rich()).AssertSubtest(t, contract("NotFoundError"),
			"carries the package prefix like a sentinel does")
	})

	t.Run("checks that a zero value reports rather than panics", func(t *testing.T) {
		t.Parallel()
		// A message dereferencing a field the zero value leaves nil crashes
		// inside whatever was already going wrong.
		rendered(t, rich()).AssertSubtest(t, contract("PlainError"),
			"reports a message for its zero value")
	})

	t.Run("checks a type against the package's other error types", func(t *testing.T) {
		t.Parallel()
		rendered(t, rich()).AssertSubtest(t, contract("NotFoundError"),
			"does not match the package's other error types")
	})

	t.Run("omits an errors.As recovery check", func(t *testing.T) {
		t.Parallel()
		// As finds a value by assignability while walking the chain, so it
		// succeeds for any type reachable through it and fails for none.
		rendered(t, rich()).AssertNoSubtest(t, contract("NotFoundError"),
			"must be recoverable with errors.As")
	})

	t.Run("checks a declared Is against errors.Is", func(t *testing.T) {
		t.Parallel()
		// An Is on the wrong receiver form is never consulted, and the type
		// then silently matches nothing.
		rendered(t, rich()).InFunc(t, contract("NotFoundError")).
			AssertContains(t, "errors.Is must agree with NotFoundError.Is")
	})

	t.Run("omits the Is check for a type declaring none", func(t *testing.T) {
		t.Parallel()
		rendered(t, rich()).AssertNoSubtest(t, contract("PlainError"),
			"agrees with errors.Is about what it matches")
	})

	t.Run("checks unwrap for a type carrying a cause", func(t *testing.T) {
		t.Parallel()
		rendered(t, rich()).InFunc(t, contract("WrappedError")).
			AssertContains(t, "WrappedError must expose its cause")
	})

	t.Run("omits the unwrap check for a type carrying none", func(t *testing.T) {
		t.Parallel()
		// Without a field to put a cause in there is nothing to hand the type,
		// so the check is dropped rather than run against a nil.
		rendered(t, rich()).AssertNoSubtest(t, contract("PlainError"),
			"exposes the cause it was given")
	})

	t.Run("writes a value into every string field it checks", func(t *testing.T) {
		t.Parallel()
		rendered(t, rich()).InFunc(t, contract("NotFoundError")).
			AssertContains(t, `Key: "test-key"`)
	})

	t.Run("ignores an unexported type declaring Error", func(t *testing.T) {
		t.Parallel()
		// Same reason as an unexported sentinel: the checks live outside the
		// package and cannot name it.
		b := rich()
		b.Struct("hiddenError", func(st *storefixture.StructBuilder) {
			st.Pos(sdk.At(fixtureFile, 1, 1))
			st.Field("Detail", storefixture.Named("string"), nil)
			st.Method(golang.MethodError, errorSig)
		})
		rendered(t, b).AssertNoFunc(t, contract("hiddenError"))
	})

	t.Run("ignores an unexported field of an error type", func(t *testing.T) {
		t.Parallel()
		// A literal naming it would not compile from the test package.
		rendered(t, rich()).InFunc(t, contract("NotFoundError")).
			AssertNotContains(t, "secret:")
	})

	t.Run("leaves a field with no recorded type out of the message check", func(t *testing.T) {
		t.Parallel()
		// A frontend that could not resolve a field's type records none, and
		// there is no sample value for a type nobody knows.
		rendered(t, rich()).InFunc(t, contract("NotFoundError")).
			AssertNotContains(t, "Untyped:")
	})

	t.Run("leaves a non-string field out of the message check", func(t *testing.T) {
		t.Parallel()
		// A message renders a number through a verb whose width and base are
		// not visible here, so asserting that "42" appears would fail against
		// %03d for a field that is perfectly well reported.
		rendered(t, rich()).InFunc(t, contract("NotFoundError")).
			AssertNotContains(t, "Attempts must reach the message")
	})

	t.Run("reads the error contract by shape rather than by name", func(t *testing.T) {
		t.Parallel()
		// A type declaring `Error()` with no result is not an error, and a
		// generated check calling it as one would not compile. Matching the
		// signature is what eidos does, and this pins that the plugin asks
		// eidos rather than asking for the name.
		b := bare("")
		b.Struct("ShoutyError", func(st *storefixture.StructBuilder) {
			st.Pos(sdk.At(fixtureFile, 1, 1))
			st.Method(golang.MethodError, nil)
		})
		rendered(t, b).AssertNoFunc(t, contract("ShoutyError"))
	})
}

// A package may carry an error type and no sentinel at all, which decides both
// what is emitted and — because the output file is named after a declaration —
// where it lands.
func TestErrorTypesWithoutSentinels(t *testing.T) {
	t.Parallel()

	t.Run("emits the per-type checks", func(t *testing.T) {
		t.Parallel()
		rendered(t, typesOnly()).AssertFunc(t, contract("PlainError"))
	})

	t.Run("emits no sentinel umbrella", func(t *testing.T) {
		t.Parallel()
		// There is no set to assert anything about, and a check over an empty
		// one would read as though the package had been examined.
		rendered(t, typesOnly()).AssertNoFunc(t, suite)
	})

	t.Run("omits the Is check with no sentinel to compare against", func(t *testing.T) {
		t.Parallel()
		// The check asks whether errors.Is reaches the declared method, and
		// with nothing to pass it there is no question to ask.
		rendered(t, typesOnly()).AssertNoSubtest(t, contract("PlainError"),
			"agrees with errors.Is about what it matches")
	})
}

// A struct's declarations are not its method set. `type NotFoundError struct{
// BaseError }` is the dominant Go idiom for a family of custom errors, and a
// generator reading only the declarations finds no Error method on it at all.
func TestErrorTypesReachedThroughEmbedding(t *testing.T) {
	t.Parallel()

	t.Run("emits the checks for the embedder", func(t *testing.T) {
		t.Parallel()
		rendered(t, promoted()).AssertFunc(t, contract("NotFoundError"))
	})

	t.Run("reads the contract off the promoted method set", func(t *testing.T) {
		t.Parallel()
		// NotFoundError declares no method at all, so a check about what its
		// Error returns can only have come from the embedded type.
		rendered(t, promoted()).AssertSubtest(t, contract("NotFoundError"),
			"reports a message for its zero value")
	})

	t.Run("anchors the file on the embedder", func(t *testing.T) {
		t.Parallel()
		// The anchor decides the output filename, and it asked a different
		// question from the one that found the type: declared methods rather
		// than promoted. A package whose only error type inherits its contract
		// then had no anchor at all, and its checks were dropped in silence.
		rendered(t, promoted()).AssertFunc(t, contract("BaseError"))
	})
}

// An embed the run did not load leaves the method set smaller than the truth,
// so generating against it asserts a contract the type may not have.
func TestUnresolvedEmbed(t *testing.T) {
	t.Parallel()

	t.Run("reports the embed it could not follow", func(t *testing.T) {
		t.Parallel()
		got := diagnostics(t, foreignEmbed())
		testkit.Len(t, got, 1, "an unresolvable embed is reported once")
	})

	t.Run("names the embed as the author wrote it", func(t *testing.T) {
		t.Parallel()
		// `io.Closer` rather than the bare `Closer` the node carries: the
		// author fixes this by loading a package, and the message has to name
		// the one they would load.
		testkit.Contains(t, diagnostics(t, foreignEmbed())[0].Message, "io.Closer",
			"the diagnostic names the embed as written")
	})

	t.Run("generates against the smaller set rather than nothing", func(t *testing.T) {
		t.Parallel()
		// A warning, not an error: the declared half of the contract is still
		// worth checking, and refusing the whole package would punish a type
		// for an import the run happened not to reach.
		rendered(t, foreignEmbed()).AssertFunc(t, contract("PartialError"))
	})
}

// The anchor decides which file the checks land in, and a run holds every
// package it loaded — so the scan has to be scoped to the one under generation.
func TestAnchorScope(t *testing.T) {
	t.Parallel()

	t.Run("ignores an error type in another package", func(t *testing.T) {
		t.Parallel()
		// Anchoring on a neighbour's declaration would compose the filename
		// from the neighbour's source basename, writing this package's checks
		// into a file named after a file it does not contain.
		//
		// The foreign package is loaded first, deliberately: the scan returns
		// its first match, so a run that saw the subject first would pass
		// against a scan with no scope check at all.
		golangtest.Rendered(t, run(t, foreignErrorType(), typesOnly().PackageNode())).
			Suffixed(t, sentinel.GoSuffix).
			AssertFunc(t, contract("PlainError"))
	})
}

// Swallowing a failed append reads downstream as a package nobody annotated
// rather than as a fault, and the checks are this generator's whole output.
func TestGenerateReportsAFailedAppend(t *testing.T) {
	t.Parallel()

	s := bare("").Build()
	// Freezing from outside the pipeline stands in for the real cause: an
	// append arriving after Layout has closed the graph.
	s.Emit().Freeze()

	err := sentinel.New().Generate(&sdk.GeneratorContext{
		Store: s, Reader: sdk.NewStoreReader(s), Diag: sdk.NewSink(),
	})

	testkit.Error(t, err, "a failed append must surface")
	testkit.Contains(t, err.Error(), fixturePkgPath, "the error must name the package")
}

// The cross-package check needs two packages by construction, so a fixture set
// with one leaves the directive parsing, naming nothing, and passing.
func TestNoOverlap(t *testing.T) {
	t.Parallel()

	t.Run("checks against the named package", func(t *testing.T) {
		t.Parallel()
		renderedWith(t, neighbouring(), neighbour()).
			AssertSubtest(t, suite, "no sentinel matches one in other")
	})

	t.Run("names the neighbour's sentinels rather than its own", func(t *testing.T) {
		t.Parallel()
		renderedWith(t, neighbouring(), neighbour()).
			InFunc(t, suite).AssertContains(t, "ErrGone")
	})
}

// The diagnostics are what a corpus cannot show: a fixture provoking one would
// fail the run that generates every other fixture.
func TestDiagnostics(t *testing.T) {
	t.Parallel()

	t.Run("reports a package with no error contract to check", func(t *testing.T) {
		t.Parallel()
		// The directive says the package's errors are a contract. A file
		// asserting nothing about an empty set would read as though they had
		// been checked.
		empty := storefixture.New().Package(fixturePkgName, fixturePkgPath)
		annotate(empty, storefixture.Directive("sentinel"))
		testkit.Len(t, diagnostics(t, empty), 1, "a package with nothing to check is reported")
	})

	t.Run("ignores a no-overlap directive naming nothing", func(t *testing.T) {
		t.Parallel()
		// The schema declares the argument, so a bare line is a malformed
		// directive rather than a package to check against.
		b := bare("")
		annotate(b, storefixture.Directive("sentinel-no-overlap-with"))
		testkit.Len(t, diagnostics(t, b), 0, "a directive naming nothing is skipped")
	})

	t.Run("reports a package declaring non-overlap with itself", func(t *testing.T) {
		t.Parallel()
		// Every sentinel matches itself, so the check would fail for a package
		// behaving exactly as intended.
		b := bare("")
		annotate(b, storefixture.Directive("sentinel-no-overlap-with",
			storefixture.Arg(fixturePkgPath)))
		testkit.Len(t, diagnostics(t, b), 1, "a self-reference is reported")
	})
}

// The golden is the readable record of what this generator produces. A diff on
// it is the review surface for any template change — the assertions above say
// a check is present, and only the golden says what the whole file reads like.
//
// Regenerate by deleting it: a missing golden is written on the next run, and
// an existing one is never rewritten, so the change stays visible in review.
func TestRenderMatchesGolden(t *testing.T) {
	t.Parallel()

	f := renderedWith(t, full(), neighbour())
	f.AssertPackage(t, fixturePkgName+"_test")
	f.AssertFormatted(t)
	f.AssertGeneratedHeader(t)
	f.AssertGolden(t, "testdata/golden/"+renderedFile)
}

// rendered drives the plugin and the Go backend over b through a synthetic
// pipeline, so routing and rendering both participate, and returns the one
// file the run produced.
func rendered(t *testing.T, b *storefixture.Builder) *golangtest.Source {
	t.Helper()
	return golangtest.Render(t, backendgolang.New(), b.PackageNode(), sentinel.New()).
		Suffixed(t, sentinel.GoSuffix)
}

// renderedWith is [rendered] over a run that also loaded other packages.
//
// The cross-package check reads a neighbour's sentinels out of the run's own
// index, so the fixture has to be two packages rather than one — which is also
// how a consumer meets it, since a neighbour is an ordinary package that
// happens to be in the pattern.
func renderedWith(t *testing.T, b *storefixture.Builder, others ...*sdk.Package) *golangtest.Source {
	t.Helper()
	pkgs := append([]*sdk.Package{b.PackageNode()}, others...)
	return golangtest.Rendered(t, run(t, pkgs...)).Suffixed(t, sentinel.GoSuffix)
}

// diagnostics drives the plugin over b and returns what the run reported.
func diagnostics(t *testing.T, b *storefixture.Builder) []sdk.Diag {
	t.Helper()
	return run(t, b.PackageNode()).Diagnostics().Diagnostics()
}

// run drives one pipeline over the given packages.
//
// [golangtest.Render] covers the single-package case in one call; this exists
// for the two that it cannot — a run over several packages, and a run whose
// subject is the diagnostics rather than the files, which must not be adopted
// because adoption fails the test on a recorded error.
func run(t *testing.T, pkgs ...*sdk.Package) *pipelinetest.Pipeline {
	t.Helper()
	return pipelinetest.New(t).
		WithFrontend(pipelinetest.FromNodes(pkgs...)).
		WithGenerator(sentinel.New()).
		WithBackend(backendgolang.New()).
		Build().
		Run("./...")
}

// contract composes a per-type check's identifier the way the template does.
func contract(typeName string) string { return golang.TestFuncName(typeName, "Contract") }

// annotate attaches a directive to the fixture's package, which is where this
// plugin's schemas are scoped.
//
// Through the builder rather than by appending to the node graph: reaching past
// the fixture API to mutate a node is how a fixture ends up depending on a
// field order the builder is free to change.
func annotate(b *storefixture.Builder, d *sdk.Directive) { b.Directive(d) }

// emptyPrefix asks [bare] for `prefix=` with no value, which is the spelling
// an author reaches for when they mean to remove the check.
const emptyPrefix = "=empty"

// pos puts a fixture variable in the file the output is named after.
func pos(v *storefixture.VariableBuilder) { v.Pos(sdk.At(fixtureFile, 1, 1)) }

// bare returns the floor: three sentinels, no error types, with prefix set to
// value when non-empty.
func bare(prefix string) *storefixture.Builder {
	b := storefixture.New().Package(fixturePkgName, fixturePkgPath)
	// Declared explicitly: the projection tracks the imports its *type*
	// expressions need, and an initialiser is opaque text it cannot read a
	// qualifier out of.
	b.Import("errors")
	for _, name := range []string{"ErrEmpty", "ErrFull", "ErrInvalid"} {
		b.Variable(name, pos)
	}
	dir := storefixture.Directive("sentinel")
	switch prefix {
	case "":
	case emptyPrefix:
		dir = storefixture.Directive("sentinel", storefixture.KV("prefix", ""))
	default:
		dir = storefixture.Directive("sentinel", storefixture.KV("prefix", prefix))
	}
	annotate(b, dir)
	return b
}

// typesOnly declares an error type and no sentinel, so the anchor the output
// is named after has to come from the type rather than from a variable.
func typesOnly() *storefixture.Builder {
	b := storefixture.New().Package(fixturePkgName, fixturePkgPath)
	b.Struct("PlainError", func(s *storefixture.StructBuilder) {
		s.Pos(sdk.At(fixtureFile, 1, 1))
		s.Field("Detail", storefixture.Named("string"), nil)
		s.Method(golang.MethodError, errorSig)
		s.Method(golang.MethodIs, isSig)
	})
	annotate(b, storefixture.Directive("sentinel"))
	return b
}

// promoted declares its whole error contract on an embedded type and the
// embedder declares nothing, which is what the idiom looks like.
//
// The embedder adds no field of its own, deliberately: a promoted Error cannot
// mention a field the embedder declared, so a type that both embeds its
// contract and adds a field asserts two things at once.
func promoted() *storefixture.Builder {
	b := storefixture.New().Package(fixturePkgName, fixturePkgPath)
	b.Struct("BaseError", func(s *storefixture.StructBuilder) {
		s.Pos(sdk.At(fixtureFile, 1, 1))
		s.Field("Cause", storefixture.Named("error"), nil)
		s.Method(golang.MethodError, errorSig)
		s.Method(golang.MethodUnwrap, unwrapSig)
	})
	b.Struct("NotFoundError", func(s *storefixture.StructBuilder) {
		s.Pos(sdk.At(fixtureFile, 1, 1))
		s.Embed(storefixture.PkgNamed(fixturePkgPath, "BaseError"))
	})
	annotate(b, storefixture.Directive("sentinel"))
	return b
}

// foreignEmbed declares an error type embedding one from a package the run did
// not load, which is the ordinary cause: the pattern reached this package and
// not the one it imports.
func foreignEmbed() *storefixture.Builder {
	b := storefixture.New().Package(fixturePkgName, fixturePkgPath)
	b.Struct("PartialError", func(s *storefixture.StructBuilder) {
		s.Pos(sdk.At(fixtureFile, 1, 1))
		s.Embed(storefixture.PkgNamed("io", "Closer"))
		s.Method(golang.MethodError, errorSig)
	})
	annotate(b, storefixture.Directive("sentinel"))
	return b
}

// foreignPath is the package [foreignErrorType] declares. Ordered before the
// fixture's own path, because the scan returns its first match and a fixture
// whose foreign candidate sorts last would pass against a scan with no scope
// check at all.
const foreignPath = "example.com/audit"

// foreignErrorType is a second package declaring an error type of its own and
// no directive, so a run holds two candidates and only one is in scope.
func foreignErrorType() *sdk.Package {
	b := storefixture.New().Package("audit", foreignPath)
	b.Struct("AuditError", func(s *storefixture.StructBuilder) {
		s.Pos(sdk.At("audit/errors.go", 1, 1))
		s.Method(golang.MethodError, errorSig)
	})
	return b.PackageNode()
}

// rich adds the three error-type shapes: one declaring Is, one declaring Unwrap
// with a cause, and one declaring neither.
func rich() *storefixture.Builder {
	b := bare("")
	b.Struct("NotFoundError", func(s *storefixture.StructBuilder) {
		s.Pos(sdk.At(fixtureFile, 1, 1))
		s.Field("Key", storefixture.Named("string"), nil)
		s.Field("Attempts", storefixture.Named("int"), nil)
		// An unexported field and one whose type the frontend could not
		// record: both are skipped, and both are here so the skip is
		// observable rather than assumed.
		s.Field("secret", storefixture.Named("string"), nil)
		s.Field("Untyped", nil, nil)
		s.Method(golang.MethodError, errorSig)
		s.Method(golang.MethodIs, isSig)
	})
	b.Struct("WrappedError", func(s *storefixture.StructBuilder) {
		s.Pos(sdk.At(fixtureFile, 1, 1))
		s.Field("Cause", storefixture.Named("error"), nil)
		s.Method(golang.MethodError, errorSig)
		s.Method(golang.MethodUnwrap, unwrapSig)
	})
	b.Struct("PlainError", func(s *storefixture.StructBuilder) {
		s.Pos(sdk.At(fixtureFile, 1, 1))
		s.Field("Detail", storefixture.Named("string"), nil)
		s.Method(golang.MethodError, errorSig)
	})
	return b
}

// full is the widest fixture: every sentinel check, every error-type shape and
// the cross-package one. It is what the golden records.
func full() *storefixture.Builder {
	b := rich()
	annotate(b, storefixture.Directive("sentinel-no-overlap-with",
		storefixture.Arg(neighbourPath)))
	return b
}

// neighbouring returns a package declaring non-overlap with another that the
// same run also loaded.
func neighbouring() *storefixture.Builder {
	b := bare("")
	annotate(b, storefixture.Directive("sentinel-no-overlap-with",
		storefixture.Arg(neighbourPath)))
	return b
}

// neighbourPath is the package [neighbour] declares, named in the directive.
const neighbourPath = "example.com/other"

// neighbour is the second package a cross-package check needs. It carries no
// directive of its own: a neighbour is opted in by being named, not by
// agreeing, which is what makes the check usable against a package the author
// does not control.
func neighbour() *sdk.Package {
	b := storefixture.New().Package("other", neighbourPath)
	b.Variable("ErrGone", func(v *storefixture.VariableBuilder) {
		v.Pos(sdk.At("other/errors.go", 1, 1))
	})
	return b.PackageNode()
}

// The signatures the error contract is matched on.
//
// Matched on shape rather than on a name — `Error() string`, `Is(error) bool`,
// `Unwrap() error` — so a fixture declaring a bare `Error` with no results is
// not an error type, and a generator that skipped it would be right. Spelling
// them out is what keeps the fixture a model of the thing under test rather
// than of the assertion.
func errorSig(m *storefixture.MethodBuilder)  { m.Return(storefixture.Named("string")) }
func unwrapSig(m *storefixture.MethodBuilder) { m.Return(storefixture.Named("error")) }
func isSig(m *storefixture.MethodBuilder) {
	m.Param("target", storefixture.Named("error"))
	m.Return(storefixture.Named("bool"))
}

// Every other assertion in this file reads the generated text. This one hands
// it to the compiler.
//
// The gap it closes is particular to this generator: its entire output is a
// test suite, so until something builds and runs that suite, nothing has
// established the checks it emits can execute at all — let alone fail. A
// contract check calling `Error()` on a type whose receiver form is wrong, or
// naming a field the struct does not have, is invisible to every substring
// assertion above and obvious to `go build`.
//
// [projectable] rather than [rich]: the two carry the same three error-type
// shapes — one declaring Is, one Unwrap with a cause, one neither — but rich
// also holds a field the frontend recorded no type for, which exists to prove
// the projection skips it and which [storefixture.Builder.GoSource] rightly
// refuses to spell. A fixture cannot both stand for unspellable input and be
// projected into the support package this assertion needs.
//
// Compiles and vets, but deliberately not [golangtest.Generated.AssertTestsPass].
// This is the one generator whose emitted checks call into behaviour it did not
// write: enum, builder and stub each generate the surface their checks drive, so
// running the suite exercises generated code, while these checks call `Error`,
// `Is` and `Unwrap` on the consumer's own types. A projected fixture reproduces
// shape and not behaviour — every method body is
// `panic("storefixture: GoSource projects shape, not behaviour")` — so running
// the suite here would assert against a panic and report this fixture's
// emptiness as a generator defect.
//
// What compiling proves is the half that can actually be wrong: the receiver
// form each check writes, the field names it names, and the cause plumbing. All
// three reach a consumer's build if the projection is wrong, and none is visible
// to a substring assertion.
//
// One test and serial. Each toolchain assertion shells out to `go`, and the
// module is built once and shared.
func TestToolchainAcceptsTheChecks(t *testing.T) {
	t.Parallel()

	gen := golangtest.Render(t, backendgolang.New(), projectable().PackageNode(), sentinel.New()).
		WithSource(golangtest.GoFile(projectable().GoSource())).
		WithRequire(sentinel.Module, filepath.Join("..", ".."))

	gen.AssertCompiles(t)
	gen.AssertVets(t)
}

// projectable is the toolchain fixture: three error-type shapes, and no
// sentinel variables.
//
// Separate from [rich] rather than derived from it, because the two are built
// for opposite purposes. rich holds a field the frontend recorded no type for,
// and [bare]'s sentinels declare neither a type nor an initialiser — both stand
// for input [storefixture.Builder.GoSource] is right to refuse, and both exist
// to prove the generator skips them. A fixture cannot both model unspellable
// input and be projected into the support package these assertions compile
// against.
//
// The sentinels carry real initialisers, so the umbrella checks are compiled
// alongside the per-type ones rather than left to the golden. That needs an
// import referenced only by an `InitExpr`, which the projection now marks.
// Their messages carry the package prefix because the generated suite
// checks for it: a sentinel initialised to `errors.New("empty")` would render a
// check that compiles and then fails, reporting this fixture's wording as a
// generator defect.
func projectable() *storefixture.Builder {
	b := storefixture.New().Package(fixturePkgName, fixturePkgPath)
	b.Import("errors")
	for _, name := range []string{"ErrEmpty", "ErrFull", "ErrInvalid"} {
		b.Variable(name, func(v *storefixture.VariableBuilder) {
			v.Pos(sdk.At(fixtureFile, 1, 1))
			v.Type(storefixture.Named("error"))
			v.InitExpr(`errors.New("` + fixturePkgName + `: ` + strings.ToLower(name[3:]) + `")`)
		})
	}
	annotate(b, storefixture.Directive("sentinel"))
	b.Struct("NotFoundError", func(s *storefixture.StructBuilder) {
		s.Pos(sdk.At(fixtureFile, 1, 1))
		s.Field("Key", storefixture.Named("string"), nil)
		s.Method(golang.MethodError, errorSig)
		s.Method(golang.MethodIs, isSig)
	})
	b.Struct("WrappedError", func(s *storefixture.StructBuilder) {
		s.Pos(sdk.At(fixtureFile, 1, 1))
		s.Field("Cause", storefixture.Named("error"), nil)
		s.Method(golang.MethodError, errorSig)
		s.Method(golang.MethodUnwrap, unwrapSig)
	})
	b.Struct("PlainError", func(s *storefixture.StructBuilder) {
		s.Pos(sdk.At(fixtureFile, 1, 1))
		s.Field("Detail", storefixture.Named("string"), nil)
		s.Method(golang.MethodError, errorSig)
	})
	return b
}
