// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package enum_test

import (
	"testing"

	backendgolang "go.thesmos.sh/eidos/backend/golang"
	"go.thesmos.sh/eidos/core/diag"
	"go.thesmos.sh/eidos/core/directive"
	"go.thesmos.sh/eidos/core/position"
	"go.thesmos.sh/eidos/eidostest/pipelinetest"
	"go.thesmos.sh/eidos/eidostest/plugintest"
	"go.thesmos.sh/eidos/eidostest/storefixture"
	"go.thesmos.sh/eidos/node"
	"go.thesmos.sh/eidos/plugin"
	"go.thesmos.sh/eidos/store"

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
				Name: "annotated enum",
				BuildStore: func(t *testing.T) *store.Store {
					t.Helper()
					return storeOf(t, numeric())
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

// A numeric enum's identifier is the only textual form its declaration
// carries; a string enum's value already is one. Deriving the wrong one still
// round-trips against itself, so nothing but this notices.
func TestTextualForm(t *testing.T) {
	t.Parallel()

	t.Run("renders a numeric variant by its identifier", func(t *testing.T) {
		t.Parallel()
		api(t, numeric()).Contains(`return "Draft"`)
	})

	t.Run("strips the type's name from the identifier", func(t *testing.T) {
		t.Parallel()
		// The type is already context wherever the value appears, so repeating
		// it is noise in every log line and wire payload.
		api(t, prefixed()).Contains(`return "Active"`)
	})

	t.Run("renders a string variant by its declared value", func(t *testing.T) {
		t.Parallel()
		// The value is the textual form and it is already written down.
		// Deriving `US` instead breaks every value arriving from JSON, a
		// column or a query parameter — while still round-tripping against
		// itself, so a check over the generated pair alone would pass.
		api(t, stringly()).Contains(`return "us-east"`)
	})

	t.Run("parses a string variant from its declared value", func(t *testing.T) {
		t.Parallel()
		api(t, stringly()).Contains(`case "us-east":`)
	})

	t.Run("falls back textually for a string enum", func(t *testing.T) {
		t.Parallel()
		// A numeric conversion does not compile for one, and rendering the
		// value itself is the more useful diagnostic: the string that failed
		// to match is exactly what the caller needs to see.
		api(t, stringly()).Contains("return string(v)")
	})

	t.Run("takes a directive override over either derivation", func(t *testing.T) {
		t.Parallel()
		api(t, overridden()).Contains(`return "Aspirin"`)
	})

	t.Run("keeps an unparseable value verbatim", func(t *testing.T) {
		t.Parallel()
		// A string variant's value arrives in source form. One the unquoter
		// refuses is malformed source, and passing it through unchanged puts
		// the author's own text in front of them rather than a guess.
		api(t, unquotable()).Contains("unterminated")
	})

	t.Run("falls back numerically for an enum with no recorded base type", func(t *testing.T) {
		t.Parallel()
		// Nothing records it as a string, so the numeric reading is the safe
		// one — and the conversion names whatever the frontend did record.
		api(t, untypedBase()).Contains("Bare(%d)")
	})
}

// unquotable declares a string value that is not a valid Go literal.
func unquotable() *node.Package {
	return fixture("Broken", "string", nil, nil, variant("Bad", `"unterminated`))
}

// untypedBase carries no underlying type, which a frontend may leave unset.
func untypedBase() *node.Package {
	pkg := fixture("Bare", "int", nil, nil, variant("One", "1"))
	pkg.Enums[0].Underlying = nil
	return pkg
}

// An author who wrote their own method meant to keep it. A generator that
// refused to run until they deleted it would be demanding they give up the
// more specific statement.
func TestExistingMethods(t *testing.T) {
	t.Parallel()

	t.Run("skips a method the type already declares", func(t *testing.T) {
		t.Parallel()
		api(t, handWritten()).NotContains("func (v Hand) String() string")
	})

	t.Run("still generates the methods it does not declare", func(t *testing.T) {
		t.Parallel()
		api(t, handWritten()).Contains("func (v Hand) IsValid() bool")
	})

	t.Run("withholds unmarshal when parse was not generated", func(t *testing.T) {
		t.Parallel()
		// UnmarshalText is written in terms of Parse, and Parse rides with
		// String. Emitted anyway it would name a function nothing declares —
		// which is a build failure in the consumer's package.
		api(t, handWritten()).NotContains("func (v *Hand) UnmarshalText")
	})

	t.Run("generates nothing at all on methods=off", func(t *testing.T) {
		t.Parallel()
		// The whole primary file is withheld: one carrying only a
		// generated-by header reads as a generator that failed.
		render(t, suppressed()).AssertNoFile("types" + enum.GoPrimarySuffix)
	})

	t.Run("still checks the declared set on methods=off", func(t *testing.T) {
		t.Parallel()
		// The set is what the directive was pointing at; only the surface was
		// declined.
		tests(t, suppressed()).Contains("TestManualVariants")
	})
}

// Which checks an enum earns depends on what exists to drive them, and a check
// naming a method the type has not got is a build failure rather than a
// finding.
func TestChecks(t *testing.T) {
	t.Parallel()

	t.Run("pins the declared arity", func(t *testing.T) {
		t.Parallel()
		tests(t, numeric()).Contains("declares exactly 3 variants")
	})

	t.Run("asserts a zero outside the set is invalid", func(t *testing.T) {
		t.Parallel()
		tests(t, numeric()).Contains("the zero value is not a declared variant")
	})

	t.Run("asserts a zero inside the set is the first variant", func(t *testing.T) {
		t.Parallel()
		// The opposite reading, and which one an enum earns is exactly what a
		// fixture pair exists to tell apart.
		tests(t, zeroDeclared()).Contains("the zero value is Low")
	})

	t.Run("probes one past the last declared value", func(t *testing.T) {
		t.Parallel()
		tests(t, numeric()).Contains("Status(4)")
	})

	t.Run("omits the round-trip checks when parse was not generated", func(t *testing.T) {
		t.Parallel()
		tests(t, handWritten()).NotContains("TestHandText")
	})

	t.Run("omits the boundary probe when no value past the set can be named", func(t *testing.T) {
		t.Parallel()
		// A float's value arrives as an exact rational this cannot render back
		// into source, so the check is dropped rather than written against a
		// value that might be declared.
		tests(t, rational()).NotContains("does not render as one inside it")
	})

	t.Run("probes past the set for a string enum too", func(t *testing.T) {
		t.Parallel()
		// A marker no sensible declaration collides with, since there is no
		// arithmetic successor to a string.
		tests(t, stringly()).Contains("__testkit_unknown__")
	})

	t.Run("drops the probe when a variant already holds the marker", func(t *testing.T) {
		t.Parallel()
		// Absurd, and cheap to be right about: probing with a value the set
		// declares would assert that a declared variant is undeclared.
		tests(t, markerCollision()).NotContains("does not render as one inside it")
	})
}

// markerCollision declares the very value the string probe would use.
func markerCollision() *node.Package {
	return fixture("Edge", "string", nil, nil,
		variant("Marker", `"__testkit_unknown__"`), variant("Other", `"other"`))
}

// rational declares a value the node model records as an exact fraction, which
// is what a float enum arrives as.
func rational() *node.Package {
	return fixture("Ratio", "float64", nil, nil, variant("Half", "1/2"), variant("Third", "1/3"))
}

// The parse sentinel's message names the package, which is what a reader needs
// first when it appears in a log beside everything else.
func TestSentinelMessage(t *testing.T) {
	t.Parallel()

	t.Run("names the package the enum lives in", func(t *testing.T) {
		t.Parallel()
		// The package rather than the type: `signal: ...` in package split
		// tells a reader nothing about where to look.
		api(t, numeric()).Contains(`"cfg: unknown Status value"`)
	})

	t.Run("falls back to the path's last segment for an unloaded package", func(t *testing.T) {
		t.Parallel()
		// A run may not hold a node for the package — the declared name and
		// the directory usually agree, and a message naming a directory beats
		// one naming nothing.
		api(t, unnamedPackage()).Contains(`"elsewhere: unknown Orphan value"`)
	})

	t.Run("falls back to the whole path when it has no segments", func(t *testing.T) {
		t.Parallel()
		api(t, rootPackage()).Contains(`"solo: unknown Orphan value"`)
	})
}

// unnamedPackage owns an enum whose package the run never loaded a node for.
func unnamedPackage() *node.Package { return orphan("example.com/elsewhere") }

// rootPackage owns one whose path carries no separator at all.
func rootPackage() *node.Package { return orphan("solo") }

// orphan builds an enum belonging to a package other than the fixture's own,
// so the name lookup misses and the path has to answer.
func orphan(path string) *node.Package {
	pkg := numeric()
	e := pkg.Enums[0]
	e.Name, e.Package = "Orphan", path
	for _, v := range e.Variants {
		v.Name = "Orphan" + v.Name
	}
	return pkg
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
		testkit.Len(t, diagnostics(t, fixture("Empty", "int", nil, nil)), 1,
			"an empty enum is reported")
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
		testkit.Contains(t, got[0].Message, "example.com/other", "the diagnostic names the package")
	})

	t.Run("ignores a same-named type from a different package", func(t *testing.T) {
		t.Parallel()
		// Two packages may each declare a Status, and one's constants say
		// nothing about the other's set.
		testkit.Len(t, diagnostics(t, unrelatedPackage()), 0, "an unrelated type is not a stray")
	})
}

// strayPackage declares a constant of the enum's type in another package.
func strayPackage() *node.Package {
	pkg := numeric()
	pkg.Constants = append(pkg.Constants, &node.Constant{
		Name:    "Archived",
		Package: "example.com/other",
		Type:    storefixture.PkgNamed("example.com/cfg", "Status"),
	})
	return pkg
}

// unrelatedPackage declares a constant of a same-named type owned elsewhere.
func unrelatedPackage() *node.Package {
	pkg := numeric()
	pkg.Constants = append(pkg.Constants,
		&node.Constant{
			Name:    "Elsewhere",
			Package: "example.com/other",
			Type:    storefixture.PkgNamed("example.com/other", "Status"),
		},
		&node.Constant{Name: "Untyped", Package: "example.com/other"},
	)
	return pkg
}

// api renders the fixture and returns its primary file.
func api(t *testing.T, pkg *node.Package) *pipelinetest.FileAssertion {
	t.Helper()
	return render(t, pkg).AssertFile("types" + enum.GoPrimarySuffix)
}

// tests renders the fixture and returns its check file.
func tests(t *testing.T, pkg *node.Package) *pipelinetest.FileAssertion {
	t.Helper()
	return render(t, pkg).AssertFile("types" + enum.GoTestSuffix)
}

// render drives the plugin and the Go backend through a synthetic pipeline, so
// routing and rendering both participate.
func render(t *testing.T, pkg *node.Package) *pipelinetest.Pipeline {
	t.Helper()
	return pipelinetest.New(t).
		WithFrontend(pipelinetest.FromNodes(pkg)).
		WithGenerator(enum.New()).
		WithBackend(backendgolang.New()).
		Build().
		Run()
}

// diagnostics drives the plugin over pkg and returns what it reported.
func diagnostics(t *testing.T, pkg *node.Package) []diag.Diag {
	t.Helper()
	s := storeOf(t, pkg)
	sink := diag.New()
	ctx := &plugin.GeneratorContext{Store: s, Reader: store.NewReader(s), Diag: sink}
	if err := enum.New().Generate(ctx); err != nil {
		t.Fatalf("Generate: %v", err)
	}
	return sink.Diagnostics()
}

// storeOf wraps a package node in a store.
func storeOf(t *testing.T, pkg *node.Package) *store.Store {
	t.Helper()
	s := store.New()
	if err := s.Nodes().AddPackage(pkg); err != nil {
		t.Fatalf("AddPackage: %v", err)
	}
	return s
}

// numeric is the integer enum whose zero is not declared.
func numeric() *node.Package {
	return fixture("Status", "int", nil, nil,
		variant("Draft", "1"), variant("Published", "2"), variant("Archived", "3"))
}

// zeroDeclared starts at zero, so its zero value is a variant.
func zeroDeclared() *node.Package {
	return fixture("Priority", "int", nil, nil, variant("Low", "0"), variant("Medium", "1"))
}

// prefixed names its variants after the type, which the derivation strips.
func prefixed() *node.Package {
	return fixture("Status", "int", nil, nil, variant("StatusActive", "1"))
}

// stringly is the string-valued enum.
func stringly() *node.Package {
	return fixture("Region", "string", nil, nil,
		variant("US", `"us-east"`), variant("EU", `"eu-west"`))
}

// overridden pins one variant's textual form with the framework's directive.
func overridden() *node.Package {
	v := variant("PillAspirin", "0")
	v.DirectiveList = []*directive.Directive{
		storefixture.Directive("value", storefixture.Arg("Aspirin")),
	}
	return fixture("Pill", "int", nil, nil, v)
}

// colliding renders two variants alike, which Parse cannot invert.
func colliding() *node.Package {
	a, b := variant("First", "0"), variant("Second", "1")
	for _, v := range []*node.EnumVariant{a, b} {
		v.DirectiveList = []*directive.Directive{
			storefixture.Directive("value", storefixture.Arg("same")),
		}
	}
	return fixture("Clash", "int", nil, nil, a, b)
}

// handWritten already declares String, so Parse and Values ride out with it.
func handWritten() *node.Package {
	return fixture("Hand", "int", []*node.Method{{Name: "String"}}, nil,
		variant("Left", "0"), variant("Right", "1"))
}

// suppressed declines every method through the directive.
func suppressed() *node.Package {
	return fixture("Manual", "int", nil,
		storefixture.Directive("enum", storefixture.KV("methods", "off")),
		variant("One", "1"), variant("Two", "2"))
}

// variant builds one declared constant.
func variant(name, value string) *node.EnumVariant {
	return &node.EnumVariant{Name: name, Value: value}
}

// fixture assembles a package holding one annotated enum.
func fixture(name, underlying string, methods []*node.Method, dir *directive.Directive,
	variants ...*node.EnumVariant,
) *node.Package {
	pkg := storefixture.New().Package("cfg", "example.com/cfg").PackageNode()
	if dir == nil {
		dir = storefixture.Directive("enum")
	}
	e := &node.Enum{
		BaseNode:   node.BaseNode{SourcePos: position.At("cfg/types.go", 1, 1)},
		Name:       name,
		Package:    pkg.Path,
		Underlying: storefixture.Named(underlying),
		Variants:   variants,
		Methods:    methods,
	}
	e.DirectiveList = []*directive.Directive{dir}
	pkg.Enums = append(pkg.Enums, e)
	return pkg
}
