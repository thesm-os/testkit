// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package prescreen_test

import (
	"strings"
	"testing"

	"go.thesmos.sh/eidos/core/directive"
	"go.thesmos.sh/eidos/eidostest/plugintest"
	"go.thesmos.sh/eidos/eidostest/storefixture"
	"go.thesmos.sh/eidos/sdk"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/generator"
	"go.thesmos.sh/testkit/generator/internal/gentest"
	"go.thesmos.sh/testkit/generator/prescreen"
)

const (
	fixturePkg  = "example.com/store"
	fixtureFile = "store/iface.go"
)

// A misspelt directive is the failure this plugin exists for: it parses, it
// stamps nothing, and the output is byte-identical to source that asked for
// nothing at all.
func TestAnnotateRejectsAnUnknownName(t *testing.T) {
	t.Parallel()

	t.Run("reports it once", func(t *testing.T) {
		t.Parallel()
		testkit.Len(t, screen(t, onInterface("sutie")), 1, "the unknown name is reported")
	})

	t.Run("names what was written", func(t *testing.T) {
		t.Parallel()
		testkit.Contains(t, screen(t, onInterface("sutie"))[0].Message, `"sutie"`,
			"the diagnostic quotes the name the author wrote")
	})

	t.Run("suggests the name it is one letter from", func(t *testing.T) {
		t.Parallel()
		// The whole value of the pre-screen over a bare rejection. `sutie` is a
		// transposition of `suite`, and a reader who could spot that unaided
		// would not have written it.
		testkit.Contains(t, screen(t, onInterface("sutie"))[0].Message, `"suite"`,
			"the suggestion names the registered directive it is closest to")
	})

	t.Run("says so plainly when nothing is close", func(t *testing.T) {
		t.Parallel()
		// Suggest declines beyond half the query's length, and a suggestion of
		// something unrelated is worse than none: it sends the reader to read
		// the docs for a directive they never wanted.
		msg := screen(t, onInterface("absolutelynotadirective"))[0].Message
		testkit.Contains(t, msg, "nothing registered is close enough",
			"an implausible typo is refused without a guess")
	})
}

// Every directive the build declares passes, which is the half a rejection
// test cannot show.
//
// A pre-screen with a narrow registry rejects correct source, and it does so in
// the most confusing way available: the run reports that a directive the tool
// documents does not exist. So the registry is driven from the same composition
// the pipeline is built from, and this holds it to accepting all of it.
func TestAnnotateAcceptsEveryDeclaredDirective(t *testing.T) {
	t.Parallel()

	schemas := generator.DirectiveSchemas()
	testkit.True(t, len(schemas) > 0, "the build declares directives to accept")

	for _, s := range schemas {
		testkit.Len(t, screen(t, onInterface(string(s.Name))), 0,
			string(s.Name)+" is registered, so it passes the screen")
	}
}

// The framework's own two pass, and they come from nowhere a plugin declares.
//
// `out` and `value` are registered by the pipeline from an unexported table, so
// a pre-screen composed only from plugin schemas rejects every routed fixture
// in the corpus — which is most of it. Named in [prescreen.New], pinned here.
func TestAnnotateAcceptsTheFrameworkDirectives(t *testing.T) {
	t.Parallel()

	for _, name := range []string{"out", "value"} {
		testkit.Len(t, screen(t, onInterface(name)), 0,
			name+" is the pipeline's own and is not a plugin's to declare")
	}
}

// A directive on a node kind testkit stamps nothing on is screened too.
//
// A typo lands where the author's finger slipped. Screening only the kinds that
// carry testkit's own directives would pass `//testkit:sutie` on a struct and
// fail it on an interface, which reads as the directive being conditionally
// supported rather than misspelt.
func TestAnnotateScreensEveryNodeKind(t *testing.T) {
	t.Parallel()

	kinds := map[string]*sdk.Store{
		"package":   onPackage("sutie"),
		"file":      onFile("sutie"),
		"interface": onInterface("sutie"),
		"method":    onMethod("sutie"),
		"struct":    onStruct("sutie"),
		"field":     onField("sutie"),
	}
	for kind, s := range kinds {
		testkit.Len(t, screen(t, s), 1, "an unknown directive on a "+kind+" is reported")
	}
}

// One diagnostic per directive, not per node the walk reaches it through.
//
// The graph walk visits a method both as the interface's child and, for some
// kinds, again through a type reference. A reader given the same typo four
// times reads the count as four mistakes.
func TestAnnotateReportsEachDirectiveOnce(t *testing.T) {
	t.Parallel()

	s := storefixture.New().
		Package("store", fixturePkg).
		File(fixtureFile, nil).
		Interface("Store", func(b *storefixture.InterfaceBuilder) {
			b.Pos(sdk.At(fixtureFile, 1, 1))
			b.Directive(storefixture.Directive("sutie"))
			b.Method("Get", func(m *storefixture.MethodBuilder) {
				m.Directive(storefixture.Directive("mixn"))
				m.Param("key", storefixture.Named("string"))
				gentest.Err(m)
			})
		}).
		Build()

	got := screen(t, s)
	testkit.Len(t, got, 2, "two typos, two diagnostics")
	joined := got[0].Message + "\n" + got[1].Message
	testkit.True(t, strings.Contains(joined, "sutie") && strings.Contains(joined, "mixn"),
		"each names its own directive")
}

// The framework suites pin the static contract — a stable name, a role, and
// declarations the caller may keep.
func TestConformance(t *testing.T) {
	t.Parallel()

	plugintest.RunSuite(t, prescreen.New(generator.DirectiveSchemas()))
}

// screen runs the pre-screen over s and returns what it reported.
func screen(t *testing.T, s *sdk.Store) []sdk.Diag {
	t.Helper()
	return plugintest.Annotate(t, prescreen.New(generator.DirectiveSchemas()), s).Diagnostics()
}

// onInterface puts one directive of that name on an interface.
func onInterface(name string) *sdk.Store {
	return storefixture.New().
		Package("store", fixturePkg).
		File(fixtureFile, nil).
		Interface("Store", func(b *storefixture.InterfaceBuilder) {
			b.Pos(sdk.At(fixtureFile, 1, 1))
			b.Directive(storefixture.Directive(directive.Name(name)))
		}).
		Build()
}

// onMethod puts one directive of that name on an interface method.
func onMethod(name string) *sdk.Store {
	return storefixture.New().
		Package("store", fixturePkg).
		File(fixtureFile, nil).
		Interface("Store", func(b *storefixture.InterfaceBuilder) {
			b.Pos(sdk.At(fixtureFile, 1, 1))
			b.Method("Get", func(m *storefixture.MethodBuilder) {
				m.Directive(storefixture.Directive(directive.Name(name)))
				gentest.Err(m)
			})
		}).
		Build()
}

// onStruct puts one directive of that name on a struct.
func onStruct(name string) *sdk.Store {
	return storefixture.New().
		Package("store", fixturePkg).
		File(fixtureFile, nil).
		Struct("Config", func(b *storefixture.StructBuilder) {
			b.Pos(sdk.At(fixtureFile, 1, 1))
			b.Directive(storefixture.Directive(directive.Name(name)))
		}).
		Build()
}

// onField puts one directive of that name on a struct field.
func onField(name string) *sdk.Store {
	return storefixture.New().
		Package("store", fixturePkg).
		File(fixtureFile, nil).
		Struct("Config", func(b *storefixture.StructBuilder) {
			b.Pos(sdk.At(fixtureFile, 1, 1))
			b.Field("Host", storefixture.Named("string"), func(f *storefixture.FieldBuilder) {
				f.Directive(storefixture.Directive(directive.Name(name)))
			})
		}).
		Build()
}

// onPackage puts one directive of that name on the package.
func onPackage(name string) *sdk.Store {
	return storefixture.New().
		Package("store", fixturePkg).
		Directive(storefixture.Directive(directive.Name(name))).
		File(fixtureFile, nil).
		Build()
}

// onFile puts one directive of that name on a file.
func onFile(name string) *sdk.Store {
	return storefixture.New().
		Package("store", fixturePkg).
		File(fixtureFile, func(b *storefixture.FileBuilder) {
			b.Directive(storefixture.Directive(directive.Name(name)))
		}).
		Build()
}
