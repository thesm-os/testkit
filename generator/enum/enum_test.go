// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package enum_test

import (
	"path/filepath"
	"strconv"
	"testing"

	backendgolang "go.thesmos.sh/eidos/backend/golang"
	"go.thesmos.sh/eidos/eidostest/golangtest"
	"go.thesmos.sh/eidos/eidostest/plugintest"
	"go.thesmos.sh/eidos/eidostest/storefixture"
	"go.thesmos.sh/eidos/lang/golang"
	"go.thesmos.sh/eidos/sdk"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/generator/enum"
)

// The framework suites pin the static contract — a stable name, a well-formed
// multi-output shape, templates that parse — none of which a fixture-driven
// test would notice.
func TestConformance(t *testing.T) {
	t.Parallel()

	t.Run("satisfies the framework contracts", func(t *testing.T) {
		t.Parallel()
		plugintest.RunSuite(t, enum.New())
	})

	t.Run("satisfies the generator contracts", func(t *testing.T) {
		t.Parallel()
		plugintest.RunGeneratorSuite(t, enum.New(), []plugintest.GeneratorFixture{
			{
				Name:       "annotated enum",
				BuildStore: func(t *testing.T) *sdk.Store { t.Helper(); return numeric().Build() },
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

// The assertion every structural claim below is a proxy for.
//
// A substring check passes against a String whose switch names a variant that
// moved, a Parse returning the wrong zero, or a check that compiles and
// asserts nothing — and the generated suite is this plugin's real contract, so
// running it is the only thing that says the surface it pins behaves.
//
// Deliberately one subtest and deliberately serial. A [golangtest.Generated]
// caches its assembled module under the [testing.TB] that first built it, so
// splitting these across subtests would leave the later ones pointed at a
// TempDir the earlier one had removed.
func TestToolchainAcceptsTheEnum(t *testing.T) {
	t.Parallel()

	gen := render(t, numeric()).
		WithSource(golangtest.GoFile(numeric().GoSource())).
		WithRequire(enum.Module, filepath.Join("..", ".."))

	gen.AssertCompiles(t)
	gen.AssertVets(t)
	gen.AssertTestsPass(t)
}

// A numeric enum's identifier is the only textual form its declaration
// carries; a string enum's value already is one. Deriving the wrong one still
// round-trips against itself, so nothing but this notices.
func TestTextualForm(t *testing.T) {
	t.Parallel()

	t.Run("renders a numeric variant by its identifier", func(t *testing.T) {
		t.Parallel()
		api(t, numeric()).InMethod(t, "Status", "String").AssertContains(t, `return "Draft"`)
	})

	t.Run("strips the type's name from the identifier", func(t *testing.T) {
		t.Parallel()
		// The type is already context wherever the value appears, so repeating
		// it is noise in every log line and wire payload.
		api(t, prefixed()).InMethod(t, "Status", "String").AssertContains(t, `return "Active"`)
	})

	t.Run("renders a string variant by its declared value", func(t *testing.T) {
		t.Parallel()
		// The value is the textual form and it is already written down.
		// Deriving `US` instead breaks every value arriving from JSON, a
		// column or a query parameter — while still round-tripping against
		// itself, so a check over the generated pair alone would pass.
		api(t, stringly()).InMethod(t, "Region", "String").AssertContains(t, `return "us-east"`)
	})

	t.Run("parses a string variant from its declared value", func(t *testing.T) {
		t.Parallel()
		api(t, stringly()).InFunc(t, "ParseRegion").AssertContains(t, `case "us-east":`)
	})

	t.Run("falls back textually for a string enum", func(t *testing.T) {
		t.Parallel()
		// A numeric conversion does not compile for one, and rendering the
		// value itself is the more useful diagnostic: the string that failed
		// to match is exactly what the caller needs to see.
		api(t, stringly()).InMethod(t, "Region", "String").AssertContains(t, "return string(v)")
	})

	t.Run("takes a directive override over either derivation", func(t *testing.T) {
		t.Parallel()
		api(t, overridden()).InMethod(t, "Pill", "String").AssertContains(t, `return "Aspirin"`)
	})

	t.Run("keeps an unparseable value verbatim", func(t *testing.T) {
		t.Parallel()
		// A string variant's value arrives in source form. One the unquoter
		// refuses is malformed source, and passing it through unchanged puts
		// the author's own text in front of them rather than a guess.
		testkit.Contains(t, text(t, api(t, unquotable())), "unterminated",
			"the author's own text survives")
	})

	t.Run("falls back numerically for an enum with no recorded base type", func(t *testing.T) {
		t.Parallel()
		// A const group with no explicit type is an untyped integer, so the
		// numeric reading is the safe one. The conversion is what makes it
		// safe: printing the value unconverted would put the verb back on the
		// type whose String is being defined.
		api(t, untypedBase()).InMethod(t, "Bare", "String").
			AssertContains(t, `fmt.Sprintf("Bare(%d)", int(v))`)
	})

	t.Run("matches the verb to what the conversion produces", func(t *testing.T) {
		t.Parallel()
		// `%d` on a float printed `%!d(float64=0.5)` and vetted as a defect in
		// the consuming repository, where nobody wrote it.
		api(t, decimal()).InMethod(t, "Ratio", "String").
			AssertContains(t, `fmt.Sprintf("Ratio(%g)", float64(v))`)
	})
}

// An author who wrote their own method meant to keep it. A generator that
// refused to run until they deleted it would be demanding they give up the
// more specific statement.
func TestExistingMethods(t *testing.T) {
	t.Parallel()

	t.Run("skips a method the type already declares", func(t *testing.T) {
		t.Parallel()
		api(t, handWritten()).AssertNoMethod(t, "Hand", "String")
	})

	t.Run("still generates the methods it does not declare", func(t *testing.T) {
		t.Parallel()
		api(t, handWritten()).AssertMethod(t, "Hand", "IsValid").Signature(t, "() bool")
	})

	t.Run("withholds unmarshal when parse was not generated", func(t *testing.T) {
		t.Parallel()
		// UnmarshalText is written in terms of Parse, and Parse rides with
		// String. Emitted anyway it would name a function nothing declares —
		// which is a build failure in the consumer's package.
		api(t, handWritten()).AssertNoMethod(t, "Hand", "UnmarshalText")
	})

	t.Run("generates nothing at all on methods=off", func(t *testing.T) {
		t.Parallel()
		// The whole primary file is withheld: one carrying only a
		// generated-by header reads as a generator that failed. The whole set
		// is pinned rather than the one absence, so a second file appearing
		// from anywhere is caught too.
		render(t, suppressed()).AssertPaths(t, sourceDir+"/types"+enum.GoTestSuffix)
	})

	t.Run("still checks the declared set on methods=off", func(t *testing.T) {
		t.Parallel()
		// The set is what the directive was pointing at; only the surface was
		// declined.
		checks(t, suppressed()).AssertFunc(t, "TestManualVariants")
	})
}

// Which checks an enum earns depends on what exists to drive them, and a check
// naming a method the type has not got is a build failure rather than a
// finding.
func TestChecks(t *testing.T) {
	t.Parallel()

	t.Run("pins the declared arity", func(t *testing.T) {
		t.Parallel()
		checks(t, numeric()).AssertSubtest(t, "TestStatusVariants", "declares exactly 3 variants")
	})

	t.Run("asserts a zero outside the set is invalid", func(t *testing.T) {
		t.Parallel()
		checks(t, numeric()).
			AssertSubtest(t, "TestStatusVariants", "the zero value is not a declared variant")
	})

	t.Run("asserts a zero inside the set is the first variant", func(t *testing.T) {
		t.Parallel()
		// The opposite reading, and which one an enum earns is exactly what a
		// fixture pair exists to tell apart.
		checks(t, zeroDeclared()).
			AssertSubtest(t, "TestPriorityVariants", "the zero value is Low")
	})

	t.Run("probes one past the last declared value", func(t *testing.T) {
		t.Parallel()
		checks(t, numeric()).InFunc(t, "TestStatusVariants").AssertContains(t, "Status(4)")
	})

	t.Run("omits the round-trip checks when parse was not generated", func(t *testing.T) {
		t.Parallel()
		checks(t, handWritten()).AssertNoFunc(t, "TestHandText")
	})

	t.Run("probes past the set for a string enum too", func(t *testing.T) {
		t.Parallel()
		// A marker no sensible declaration collides with, since there is no
		// arithmetic successor to a string.
		checks(t, stringly()).InFunc(t, "TestRegionText").AssertContains(t, unknownText(t))
	})

	t.Run("drops the probe when a variant already holds the marker", func(t *testing.T) {
		t.Parallel()
		// Absurd, and cheap to be right about: probing with a value the set
		// declares would assert that a declared variant is undeclared.
		checks(t, markerCollision(t)).AssertNoSubtest(t, "TestEdgeText", boundaryProbe)
	})
}

// boundaryProbe is the subtest a derivable out-of-range value earns, named
// here because three tests turn on whether it is present.
const boundaryProbe = "a value outside the set does not render as one inside it"

// A float enum earns a boundary probe where its values can be read, and does
// not where they deliberately cannot.
//
// The split is the whole subject. A float constant reaches the node model two
// ways: `go/constant` renders an exactly-representable value as a rational, and
// one that is not stays in the decimal form the author typed.
//
//   - Decimal spellings derive. [golang.OutOfRangeFloat] takes the largest
//     value plus one, with no walk down for a gap: a float set cannot exhaust
//     its type, so there is no saturation case an integer set has to step
//     around.
//   - The exact-rational spelling is refused. [golang.ParseFloatValue] declines
//     `1/2` rather than reading it as 0.5, because Go's constant arithmetic
//     distinguishes the two readings and picking one silently is how a probe
//     ends up naming a value the set declares.
//
// Both are covered because the refusal is a decision, not a gap, and a test
// asserting only the working half would let it quietly become one.
func TestFloatEnum(t *testing.T) {
	t.Parallel()

	t.Run("derives the boundary for a float declared as a decimal literal", func(t *testing.T) {
		t.Parallel()
		checks(t, decimal()).AssertSubtest(t, "TestRatioText", boundaryProbe)
	})

	t.Run("omits the boundary for a float declared as an exact rational", func(t *testing.T) {
		t.Parallel()
		// Dropping the subtest is the conservative answer where the value
		// cannot be read: a probe rendered from a guess the set turned out to
		// declare would assert that a declared variant is undeclared.
		checks(t, rational()).AssertNoSubtest(t, "TestRatioText", boundaryProbe)
	})

	t.Run("still pins everything that does not need a boundary", func(t *testing.T) {
		t.Parallel()
		// The gap costs one subtest, not the file: the arity, the distinctness
		// and the round trip are all derivable without naming a value outside
		// the set.
		checks(t, decimal()).AssertSubtest(t, "TestRatioVariants", "declares exactly 2 variants")
	})

	t.Run("renders a float enum the toolchain accepts", func(t *testing.T) {
		t.Parallel()
		// `go vet` is what a consumer would have heard the format-verb defect
		// from, in their own repository, about a file they did not write. One
		// subtest and serial, for the reason [TestToolchainAcceptsTheEnum]
		// gives.
		gen := render(t, decimal()).
			WithSource(golangtest.GoFile(decimal().GoSource())).
			WithRequire(enum.Module, filepath.Join("..", ".."))

		gen.AssertCompiles(t)
		gen.AssertVets(t)
		gen.AssertTestsPass(t)
	})
}

// unknownText is the marker [golang.OutOfRangeText] probes a string enum with.
//
// Read back from the function rather than spelled out here: eidos keeps the
// marker private, and a copy of it would go on passing on the day upstream
// changes it — asserting that a probe exists while the generated file carried
// a different one.
func unknownText(t *testing.T) string {
	t.Helper()
	probe := storefixture.New()
	probe.Enum("Probe", func(e *storefixture.EnumBuilder) {
		e.Underlying(storefixture.Named("string"))
		e.Variant("Any", `"any"`)
	})
	text, ok := golang.OutOfRangeText(probe.PackageNode().Enums[0])
	testkit.True(t, ok, "a non-empty set yields a marker")
	return text
}

// The parse sentinel's message names the package, which is what a reader needs
// first when it appears in a log beside everything else.
func TestSentinelMessage(t *testing.T) {
	t.Parallel()

	t.Run("names the package the enum lives in", func(t *testing.T) {
		t.Parallel()
		// The package rather than the type: `signal: ...` in package split
		// tells a reader nothing about where to look.
		api(t, numeric()).InVar(t, "ErrUnknownStatus").
			AssertContains(t, `"cfg: unknown Status value"`)
	})

	t.Run("takes the declared name over the directory", func(t *testing.T) {
		t.Parallel()
		// The two usually agree and occasionally do not. A message naming a
		// package that does not exist sends a reader to a directory whose
		// import path they then cannot find.
		api(t, renamedPackage()).InVar(t, "ErrUnknownStatus").
			AssertContains(t, `"config: unknown Status value"`)
	})
}

// The two files carry different obligations to a consumer: the API is read and
// linted like hand-written code, and the checks have to reach a package they
// deliberately cannot see unqualified.
func TestFileShape(t *testing.T) {
	t.Parallel()

	t.Run("marks the API as generated", func(t *testing.T) {
		t.Parallel()
		api(t, numeric()).AssertGeneratedHeader(t).AssertFormatted(t)
	})

	t.Run("documents every exported declaration", func(t *testing.T) {
		t.Parallel()
		// Generated code is read by the consumer deciding whether to call it,
		// and linted by whatever their project runs.
		api(t, numeric()).AssertDocumented(t)
	})

	t.Run("imports only what the API renders", func(t *testing.T) {
		t.Parallel()
		// The sentinel and the numeric fallback, and nothing of testkit's: the
		// API is production code a consumer ships, not a test.
		api(t, numeric()).AssertImportsOnly(t, "errors", "fmt")
	})

	t.Run("lands the checks in the external test package", func(t *testing.T) {
		t.Parallel()
		// The framework keys the shift off the `_test.go` ending. Landing in
		// the enum's own package instead would stop the checks driving it the
		// way a consumer does.
		checks(t, numeric()).AssertPackage(t, "cfg_test").AssertFormatted(t)
	})

	t.Run("qualifies everything the checks reach for", func(t *testing.T) {
		t.Parallel()
		checks(t, numeric()).AssertImportsOnly(t, "testing", "example.com/cfg", enum.Module)
	})

	t.Run("runs every check in parallel", func(t *testing.T) {
		t.Parallel()
		// A generated suite lands in a consumer's package and is run on every
		// commit; one that serialises makes the generator the reason their
		// suite got slower.
		checks(t, numeric()).AssertParallel(t, "TestStatusVariants")
	})
}

// The diagnostics are what a corpus cannot show: a fixture provoking one would
// fail the run that generates every other fixture.
func TestDiagnostics(t *testing.T) {
	t.Parallel()

	t.Run("reports two variants rendering alike", func(t *testing.T) {
		t.Parallel()
		// Parse maps text back to exactly one variant, so a collision makes
		// one unreachable through it — and the generated round-trip check
		// would fail with no indication of the cause.
		testkit.Len(t, diagnostics(t, colliding()), 1, "a textual collision is reported")
	})

	t.Run("reports an enum with no variant", func(t *testing.T) {
		t.Parallel()
		// The directive points at a set. Generating a String over an empty one
		// produces a method that only ever returns its fallback.
		testkit.Len(t, diagnostics(t, empty()), 1, "an empty enum is reported")
	})

	t.Run("reports a variant declared in another package", func(t *testing.T) {
		t.Parallel()
		// Legal Go, and silently wrong: the constant is invisible to the set,
		// so IsValid would reject a value someone declared and the arity check
		// would pin a count that is not the truth.
		testkit.Len(t, diagnostics(t, strayPackage()), 1, "a stray variant is reported")
	})

	t.Run("names the package the stray variant came from", func(t *testing.T) {
		t.Parallel()
		// The author has to go somewhere to fix it, and the diagnostic is the
		// only thing that knows where.
		got := diagnostics(t, strayPackage())
		testkit.Contains(t, got[0].Message, strayPkgPath, "the diagnostic names the package")
	})

	t.Run("ignores a same-named type from a different package", func(t *testing.T) {
		t.Parallel()
		// Two packages may each declare a Status, and one's constants say
		// nothing about the other's set.
		testkit.Len(t, diagnostics(t, unrelatedPackage()), 0, "an unrelated type is not a stray")
	})
}

// An enum is opted in by the directive, and a package generally declares more
// enumerated types than the ones a surface was asked for.
func TestUndirectedEnum(t *testing.T) {
	t.Parallel()

	t.Run("generates nothing for it", func(t *testing.T) {
		t.Parallel()
		// A generator walking every enum would emit a String, a Parse and a
		// validity check for types whose author never asked for them — and
		// would collide with whatever they wrote by hand.
		api(t, mixedDirection()).AssertNoFunc(t, "ParsePlain")
	})

	t.Run("still generates for its annotated neighbour", func(t *testing.T) {
		t.Parallel()
		// The walk has to pass over an undirected enum rather than stop at it.
		api(t, mixedDirection()).AssertNoFunc(t, "ParsePlain")
		api(t, mixedDirection()).AssertFunc(t, "ParseStatus")
	})
}

// Swallowing a failed append reads downstream as an enum nobody annotated
// rather than as a fault, and the surface is this generator's whole output.
func TestGenerateReportsAFailedAppend(t *testing.T) {
	t.Parallel()

	s := numeric().Build()
	// Freezing from outside the pipeline stands in for the real cause: an
	// append arriving after Layout has closed the graph.
	s.Emit().Freeze()

	err := enum.New().Generate(&sdk.GeneratorContext{
		Store: s, Reader: sdk.NewStoreReader(s), Diag: sdk.NewSink(),
	})

	testkit.Error(t, err, "a failed append must surface")
	testkit.Contains(t, err.Error(), "Status", "the error must name the declaration")
}

// The goldens are the readable record of what this generator produces. A diff
// on them is the review surface for any template change — the assertions above
// say a construct is present, and only the golden says what the whole file
// reads like.
//
// Regenerate by deleting the file, which keeps the change visible in review.
func TestRenderMatchesGolden(t *testing.T) {
	t.Parallel()

	t.Run("the API", func(t *testing.T) {
		t.Parallel()
		api(t, numeric()).AssertGolden(t, filepath.Join("testdata", "golden", primaryFile))
	})

	t.Run("the checks", func(t *testing.T) {
		t.Parallel()
		checks(t, numeric()).AssertGolden(t, filepath.Join("testdata", "golden", checksFile))
	})
}

// The rendered filenames, composed by Layout from the source basename and the
// adapter's suffixes. Every fixture declares its enum in `cfg/types.go`, so the
// run resolves both targets into that directory.
const (
	sourceDir   = "cfg"
	sourcePath  = sourceDir + "/types.go"
	primaryFile = "types" + enum.GoPrimarySuffix
	checksFile  = "types" + enum.GoTestSuffix
)

// render drives the plugin and the Go backend over the fixture and adopts the
// files the run produced, so routing and rendering both participate.
func render(t *testing.T, b *storefixture.Builder) *golangtest.Generated {
	t.Helper()
	return golangtest.Render(t, backendgolang.New(), b.PackageNode(), enum.New())
}

// api parses the surface this run wrote.
//
// Addressed by the adapter's own suffix rather than by path: Layout composes
// the rest of the name from a source basename and routes it into the source's
// directory, neither of which this plugin declares.
func api(t *testing.T, b *storefixture.Builder) *golangtest.Source {
	t.Helper()
	return render(t, b).Suffixed(t, enum.GoPrimarySuffix)
}

// checks parses the check file this run wrote.
func checks(t *testing.T, b *storefixture.Builder) *golangtest.Source {
	t.Helper()
	return render(t, b).Suffixed(t, enum.GoTestSuffix)
}

// text returns a rendered file as a string, for a claim no scoped assertion
// can make — a value the generator passes through verbatim rather than
// declaring.
func text(t *testing.T, src *golangtest.Source) string {
	t.Helper()
	return string(src.Bytes())
}

// diagnostics drives the plugin over the fixture and returns what it reported.
//
// Through [golangtest.Driver] rather than [golangtest.Render]: every fixture
// here provokes an error, and adopting the output stops the test before the
// diagnostic can be read.
func diagnostics(t *testing.T, b *storefixture.Builder) []sdk.Diag {
	t.Helper()
	run := golangtest.Driver(t, backendgolang.New(), b.PackageNode(), enum.New()).
		Build().
		Run("./...")
	return run.Diagnostics().Diagnostics()
}

// numeric is the integer enum whose zero is not declared.
func numeric() *storefixture.Builder {
	return fixture("Status", "int", annotated, func(e *storefixture.EnumBuilder) {
		e.Variant("Draft", "1").Variant("Published", "2").Variant("Archived", "3")
	})
}

// zeroDeclared starts at zero, so its zero value is a variant.
func zeroDeclared() *storefixture.Builder {
	return fixture("Priority", "int", annotated, func(e *storefixture.EnumBuilder) {
		e.Variant("Low", "0").Variant("Medium", "1")
	})
}

// prefixed names its variants after the type, which the derivation strips.
func prefixed() *storefixture.Builder {
	return fixture("Status", "int", annotated, func(e *storefixture.EnumBuilder) {
		e.Variant("StatusActive", "1")
	})
}

// stringly is the string-valued enum.
func stringly() *storefixture.Builder {
	return fixture("Region", "string", annotated, func(e *storefixture.EnumBuilder) {
		e.Variant("US", `"us-east"`).Variant("EU", `"eu-west"`)
	})
}

// unquotable declares a string value that is not a valid Go literal.
func unquotable() *storefixture.Builder {
	return fixture("Broken", "string", annotated, func(e *storefixture.EnumBuilder) {
		e.Variant("Bad", `"unterminated`)
	})
}

// untypedBase carries no underlying type, which a frontend may leave unset.
func untypedBase() *storefixture.Builder {
	return fixture("Bare", "", annotated, func(e *storefixture.EnumBuilder) {
		e.Variant("One", "1")
	})
}

// rational declares values in the exact-fraction form `go/constant` reports an
// exactly-representable float as.
func rational() *storefixture.Builder {
	return fixture("Ratio", "float64", annotated, func(e *storefixture.EnumBuilder) {
		e.Variant("Half", "1/2").Variant("Third", "1/3")
	})
}

// decimal declares the same enum in the form its author typed.
func decimal() *storefixture.Builder {
	return fixture("Ratio", "float64", annotated, func(e *storefixture.EnumBuilder) {
		e.Variant("Half", "0.5").Variant("Third", "0.33")
	})
}

// overridden pins one variant's textual form with the framework's directive.
func overridden() *storefixture.Builder {
	return fixture("Pill", "int", annotated, func(e *storefixture.EnumBuilder) {
		e.Variant("PillAspirin", "0")
		pin(e, "PillAspirin", "Aspirin")
	})
}

// colliding renders two variants alike, which Parse cannot invert.
func colliding() *storefixture.Builder {
	return fixture("Clash", "int", annotated, func(e *storefixture.EnumBuilder) {
		e.Variant("First", "0").Variant("Second", "1")
		pin(e, "First", "same")
		pin(e, "Second", "same")
	})
}

// markerCollision declares the very value the string probe would use.
func markerCollision(t *testing.T) *storefixture.Builder {
	t.Helper()
	marker := strconv.Quote(unknownText(t))
	return fixture("Edge", "string", annotated, func(e *storefixture.EnumBuilder) {
		e.Variant("Marker", marker).Variant("Other", `"other"`)
	})
}

// handWritten already declares String, so Parse and Values ride out with it.
//
// Declared with its real signature rather than by name alone: the projection
// that stands in for the hand-written package spells whatever the fixture
// says, and a `String()` returning nothing would not be a Stringer for any
// consumer of it.
func handWritten() *storefixture.Builder {
	return fixture("Hand", "int", annotated, func(e *storefixture.EnumBuilder) {
		e.Variant("Left", "0").Variant("Right", "1")
		e.Node().Methods = append(e.Node().Methods, &sdk.Method{
			Name:    golang.MethodString,
			Returns: []*sdk.Return{{Type: storefixture.Named("string")}},
		})
	})
}

// suppressed declines every method through the directive.
func suppressed() *storefixture.Builder {
	return fixture("Manual", "int", methodsOff, func(e *storefixture.EnumBuilder) {
		e.Variant("One", "1").Variant("Two", "2")
	})
}

// empty carries the directive and no variant for it to point at.
func empty() *storefixture.Builder { return fixture("Empty", "int", annotated) }

// renamedPackage declares its enum in a directory the package is not named
// after — `example.com/cfg` holding `package config`.
//
// Set on the node rather than through [storefixture.Builder.Package], which
// retargets every declaration's synthetic source file along with the name and
// would move the output out of `cfg/`.
func renamedPackage() *storefixture.Builder {
	b := numeric()
	b.PackageNode().Name = "config"
	return b
}

// strayPkgPath owns the constants a fixture plants outside the enum's package.
const strayPkgPath = "example.com/other"

// strayPackage declares a constant of the enum's type in another package.
//
// Planted on the fixture's own package node and then repointed, because a
// [storefixture.Builder] holds exactly one package and the run has to see both
// the enum and the stray constant in one store.
func strayPackage() *storefixture.Builder {
	b := numeric()
	b.Constant("Archived", func(c *storefixture.ConstantBuilder) {
		c.Type(storefixture.PkgNamed("example.com/cfg", "Status"))
		c.Node().Package = strayPkgPath
	})
	return b
}

// unrelatedPackage declares a constant of a same-named type owned elsewhere.
func unrelatedPackage() *storefixture.Builder {
	b := numeric()
	b.Constant("Elsewhere", func(c *storefixture.ConstantBuilder) {
		c.Type(storefixture.PkgNamed(strayPkgPath, "Status"))
		c.Node().Package = strayPkgPath
	})
	b.Constant("Untyped", func(c *storefixture.ConstantBuilder) {
		c.Node().Package = strayPkgPath
	})
	return b
}

// mixedDirection declares an annotated enum beside one carrying no directive,
// which is what an ordinary package looks like.
func mixedDirection() *storefixture.Builder {
	b := numeric()
	b.Enum("Plain", func(e *storefixture.EnumBuilder) {
		e.Pos(sdk.At(sourcePath, 1, 1))
		e.Underlying(storefixture.Named("int"))
		e.Variant("One", "1")
	})
	return b
}

// annotated attaches the plain directive every fixture is opted in by.
func annotated(e *storefixture.EnumBuilder) { e.Directive(storefixture.Directive("enum")) }

// methodsOff attaches the directive that declines the whole method surface.
func methodsOff(e *storefixture.EnumBuilder) {
	e.Directive(storefixture.Directive(enum.DirectiveName,
		storefixture.KV(enum.MethodsKey, enum.MethodsOff)))
}

// pin attaches the framework's `value` directive to one variant.
//
// Reached through the node because [storefixture.EnumBuilder] offers no
// per-variant hook: Variant takes a name and a value and nothing else, so an
// authored text override has no builder spelling.
func pin(e *storefixture.EnumBuilder, variant, text string) {
	for _, v := range e.Node().Variants {
		if v.Name == variant {
			v.DirectiveList = append(v.DirectiveList,
				storefixture.Directive(golang.EnumValueDirective, storefixture.Arg(text)))
		}
	}
}

// fixture assembles a package holding one annotated enum, returning the builder
// rather than the node so a test can project the hand-written package the
// generated output references from the same declaration that drove the run.
//
// An empty underlying type leaves the enum typeless, which is a shape a
// frontend may produce and the numeric fallback has to answer for.
func fixture(name, underlying string, opts ...func(*storefixture.EnumBuilder)) *storefixture.Builder {
	b := storefixture.New().Package(sourceDir, "example.com/cfg")
	b.Enum(name, func(e *storefixture.EnumBuilder) {
		e.Pos(sdk.At(sourcePath, 1, 1))
		if underlying != "" {
			e.Underlying(storefixture.Named(underlying))
		}
		for _, opt := range opts {
			opt(e)
		}
	})
	return b
}

// The generated file names the package its subject lives in, and a declaration
// can outlive the record of where it came from.
func TestPackageNameFallback(t *testing.T) {
	t.Parallel()

	t.Run("derives the name from the path when the run recorded no package", func(t *testing.T) {
		t.Parallel()
		// A package clause differs from a path's last segment often enough that
		// the loaded node is asked first — `example.com/cfg/v2` is package
		// `cfg`. But a declaration whose package this run never recorded still
		// has to name something, and the path is the only thing left that
		// knows.
		//
		// Asserted on the projection rather than on a rendered file: routing
		// composes an output package from the same record, so a fixture missing
		// one has nothing to render into.
		testkit.Equal(t, apiOf(t, unloadedPackage()).PackageName, "unloaded",
			"the path answers where the package node cannot")
	})
}

// apiOf drives the plugin over b and returns the queued API projection.
func apiOf(t *testing.T, b *storefixture.Builder) *enum.API {
	t.Helper()
	s := b.Build()
	plugintest.Generate(t, enum.New(), s)
	for _, p := range s.Emit().PendingOriginSlots() {
		if a, ok := p.Item.(*enum.API); ok {
			return a
		}
	}
	t.Fatal("the run queued no enum API")
	return nil
}

// unloadedPackage points its enum at a package path the run never recorded,
// which is what a partial load leaves behind.
func unloadedPackage() *storefixture.Builder {
	b := numeric()
	for _, e := range b.PackageNode().Enums {
		e.Package = "example.com/unloaded"
	}
	return b
}
