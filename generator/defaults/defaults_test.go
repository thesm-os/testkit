// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package defaults_test

import (
	"testing"

	"go.thesmos.sh/eidos/eidostest/plugintest"
	"go.thesmos.sh/eidos/eidostest/storefixture"
	"go.thesmos.sh/eidos/sdk"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/generator/defaults"
	"go.thesmos.sh/testkit/generator/internal/stamp"
)

// The stamp is what every reader of a declared default sees, and the literal
// travels verbatim — so what is under test is that it arrives unaltered, and
// that the one thing which cannot travel is refused.
func TestAnnotate(t *testing.T) {
	t.Parallel()

	t.Run("carries a string literal with its quotes", func(t *testing.T) {
		t.Parallel()
		// The stamp renders straight into a composite literal, so a value that
		// lost its quoting would emerge as an identifier.
		testkit.Equal(t, stamped(t, `"localhost"`), `"localhost"`, "the literal travels verbatim")
	})

	t.Run("carries a keyword literal", func(t *testing.T) {
		t.Parallel()
		testkit.Equal(t, stamped(t, "true"), "true", "a keyword needs no quoting")
	})

	t.Run("carries an explicit zero", func(t *testing.T) {
		t.Parallel()
		// A generator reading a bare zero cannot tell "seed this to zero" from
		// "no default given", and emits the same constructor either way.
		testkit.Equal(t, stamped(t, "0"), "0", "an explicit zero is not an absence")
	})

	t.Run("carries an untyped nil", func(t *testing.T) {
		t.Parallel()
		testkit.Equal(t, stamped(t, "nil"), "nil", "nil has no literal form of its own")
	})

	t.Run("leaves an undeclared field alone", func(t *testing.T) {
		t.Parallel()
		testkit.Equal(t, stamped(t, ""), "", "a field with no directive stamps nothing")
	})

	t.Run("takes the last of two declarations", func(t *testing.T) {
		t.Parallel()
		s := fixture(t, "8080", "9090")
		annotate(t, s)
		testkit.Equal(t, read(t, s), "9090", "the last value written wins")
	})
}

// A literal that does not parse would reach a template unexamined and emerge
// as a syntax error in generated code, blamed on the generator rather than on
// the line that caused it.
func TestAnnotateRejectsAMalformedLiteral(t *testing.T) {
	t.Parallel()

	t.Run("reports it", func(t *testing.T) {
		t.Parallel()
		s := fixture(t, `"unterminated`)
		testkit.Len(t, annotate(t, s), 1, "a malformed literal is reported once")
	})

	t.Run("names the field", func(t *testing.T) {
		t.Parallel()
		s := fixture(t, `"unterminated`)
		testkit.Contains(t, annotate(t, s)[0].Message, "Config.Host", "the diagnostic names the field")
	})

	t.Run("stamps nothing", func(t *testing.T) {
		t.Parallel()
		// Dropping rather than guessing: a stamp nobody can render is worse
		// than an absent one, because it fails in generated code.
		s := fixture(t, `"unterminated`)
		annotate(t, s)
		testkit.Equal(t, read(t, s), "", "a refused literal leaves no stamp")
	})

	t.Run("reports an unterminated rune", func(t *testing.T) {
		t.Parallel()
		// The quote a hand-rolled check forgets, because the two string forms
		// are what come to mind. It swallows the rest of the generated line
		// exactly as an unbalanced double quote does.
		s := fixture(t, `'a`)
		testkit.Len(t, annotate(t, s), 1, "an unterminated rune is a malformed literal")
	})
}

// The framework suites pin the static contract — a stable name, a unique
// directive schema, declarations the caller may keep.
func TestConformance(t *testing.T) {
	t.Parallel()

	plugintest.RunSuite(t, defaults.New())
}

// The resolved import path travels beside the symbol, because a rendered file
// has to register the import and only a reference carries one.
func TestAnnotateStampsTheResolvedPackage(t *testing.T) {
	t.Parallel()

	t.Run("a qualified default names its import", func(t *testing.T) {
		t.Parallel()
		s := imported(t, "", "example.com/seed", "seed.Region")
		annotate(t, s)
		testkit.Equal(t, stamp.DefaultPackage(fieldOf(t, s).Meta()), "example.com/seed",
			"the import path is stamped")
	})

	t.Run("a plain literal names none", func(t *testing.T) {
		t.Parallel()
		s := fixture(t, `"localhost"`)
		annotate(t, s)
		testkit.Equal(t, stamp.DefaultPackage(fieldOf(t, s).Meta()), "", "a literal needs no import")
	})
}

// fieldOf returns the fixture's only field.
func fieldOf(t *testing.T, s *sdk.Store) *sdk.Field {
	t.Helper()
	for _, st := range s.Nodes().Structs().Items() {
		for _, f := range st.Fields {
			return f
		}
	}
	t.Fatal("fixture has no field")
	return nil
}

// fixturePkg is the import path every fixture here declares its struct in.
const fixturePkg = "example.com/cfg"

// fixtureFile is the file every fixture positions its struct in. Named
// explicitly rather than left to the builder's synthesised position, because a
// qualifier resolves against a file's import block and a declaration whose
// position names no declared file has no imports in scope.
const fixtureFile = "types.go"

// imported returns a store whose file imports path — under alias when one is
// given — optionally declaring values as the field's defaults.
func imported(t *testing.T, alias, path string, values ...string) *sdk.Store {
	t.Helper()
	return storefixture.New().
		Package("cfg", fixturePkg).
		File(fixtureFile, func(f *storefixture.FileBuilder) {
			f.ImportAs(alias, path)
		}).
		Struct("Config", func(b *storefixture.StructBuilder) {
			b.Pos(sdk.At(fixtureFile, 1, 1))
			b.Field("Host", storefixture.Named("string"), func(f *storefixture.FieldBuilder) {
				f.Pos(sdk.At(fixtureFile, 2, 1))
				for _, v := range values {
					f.Directive(storefixture.Directive("default", storefixture.Arg(v)))
				}
			})
		}).
		Build()
}

// stamped annotates a one-field struct declaring value and returns the stamp.
func stamped(t *testing.T, value string) string {
	t.Helper()
	s := fixture(t, value)
	annotate(t, s)
	return read(t, s)
}

// fixture returns a store whose Config.Host declares each of values in order.
// An empty slice leaves the field undeclared. The file it declares imports
// nothing, which is what the literal and full-path arms resolve against.
func fixture(t *testing.T, values ...string) *sdk.Store {
	t.Helper()
	return storefixture.New().
		Package("cfg", fixturePkg).
		File(fixtureFile, nil).
		Struct("Config", func(b *storefixture.StructBuilder) {
			b.Pos(sdk.At(fixtureFile, 1, 1))
			b.Field("Host", storefixture.Named("string"), func(f *storefixture.FieldBuilder) {
				f.Pos(sdk.At(fixtureFile, 2, 1))
				for _, v := range values {
					if v == "" {
						continue
					}
					f.Directive(storefixture.Directive("default", storefixture.Arg(v)))
				}
			})
		}).
		Build()
}

// annotate runs the plugin over s and returns what it reported.
func annotate(t *testing.T, s *sdk.Store) []sdk.Diag {
	t.Helper()
	return plugintest.Annotate(t, defaults.New(), s).Diagnostics()
}

// read returns the stamp on Config.Host.
func read(t *testing.T, s *sdk.Store) string {
	t.Helper()
	for _, st := range s.Nodes().Structs().Items() {
		for _, f := range st.Fields {
			if f.Name == "Host" {
				return stamp.DefaultOf(f.Meta())
			}
		}
	}
	t.Fatal("fixture has no Config.Host")
	return ""
}
