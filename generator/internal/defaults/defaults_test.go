// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package defaults_test

import (
	"testing"

	"go.thesmos.sh/eidos/core/diag"
	"go.thesmos.sh/eidos/core/position"
	"go.thesmos.sh/eidos/eidostest/plugintest"
	"go.thesmos.sh/eidos/eidostest/storefixture"
	"go.thesmos.sh/eidos/node"
	"go.thesmos.sh/eidos/plugin"
	"go.thesmos.sh/eidos/store"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/generator/internal/defaults"
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
}

// The framework suites pin the static contract — a stable name, a unique
// directive schema, declarations the caller may keep.
func TestConformance(t *testing.T) {
	t.Parallel()

	plugintest.RunSuite(t, defaults.New())
}

// A default naming a symbol elsewhere needs an import path, and a qualifier
// carries only a package name. Both notations exist because an import written
// solely to feed a directive is an unused import, which does not compile.
func TestResolve(t *testing.T) {
	t.Parallel()

	t.Run("passes a literal through untouched", func(t *testing.T) {
		t.Parallel()
		pkg, sym := resolve(t, `"localhost"`)
		testkit.Equal(t, pkg, "", "a literal names no package")
		testkit.Equal(t, sym, `"localhost"`, "the literal travels verbatim")
	})

	t.Run("passes a number through untouched", func(t *testing.T) {
		t.Parallel()
		// A float holds a dot without naming anything.
		pkg, sym := resolve(t, "0.75")
		testkit.Equal(t, pkg, "", "a number names no package")
		testkit.Equal(t, sym, "0.75", "the number travels verbatim")
	})

	t.Run("takes a full import path as itself", func(t *testing.T) {
		t.Parallel()
		pkg, sym := resolve(t, "example.com/seed.Region")
		testkit.Equal(t, pkg, "example.com/seed", "the path needs no lookup")
		testkit.Equal(t, sym, "Region", "the symbol is the last segment")
	})

	t.Run("splits a path whose own last segment holds a dot", func(t *testing.T) {
		t.Parallel()
		// Splitting anywhere but the last dot would cut `yaml.v3` in half.
		pkg, sym := resolve(t, "gopkg.in/yaml.v3.Marshal")
		testkit.Equal(t, pkg, "gopkg.in/yaml.v3", "the version stays with the path")
		testkit.Equal(t, sym, "Marshal", "the symbol is what follows the last dot")
	})

	t.Run("resolves a qualifier against the file's imports", func(t *testing.T) {
		t.Parallel()
		// This is the form an author writes for a package the file already
		// uses; the full-path form exists for one it does not.
		s := imported(t, "seed", "example.com/seed")
		pkg, sym, err := defaults.Resolve(readerOf(t, s), fieldOf(t, s), "seed.Region")
		testkit.NoError(t, err, "an imported qualifier resolves")
		testkit.Equal(t, pkg, "example.com/seed", "the path comes from the import")
		testkit.Equal(t, sym, "Region", "the symbol follows the qualifier")
	})

	t.Run("stamps the resolved package alongside the symbol", func(t *testing.T) {
		t.Parallel()
		// A rendered file has to register the import, which only a reference
		// carries — so the path travels beside the symbol rather than inside it.
		s := imported(t, "seed", "example.com/seed", "seed.Region")
		annotate(t, s)
		testkit.Equal(t, defaults.Package(fieldOf(t, s).Meta()), "example.com/seed",
			"the import path is stamped")
	})

	t.Run("stamps no package for a plain literal", func(t *testing.T) {
		t.Parallel()
		s := fixture(t, `"localhost"`)
		annotate(t, s)
		testkit.Equal(t, defaults.Package(fieldOf(t, s).Meta()), "", "a literal needs no import")
	})

	t.Run("reads nothing from an absent bag", func(t *testing.T) {
		t.Parallel()
		// Meta is allocated on first write, so a node nothing stamped has no
		// bag at all — and both accessors are called on every field.
		testkit.Equal(t, defaults.Of(nil), "", "an unstamped field declares no default")
		testkit.Equal(t, defaults.Package(nil), "", "and names no package")
	})

	t.Run("passes a negative number through untouched", func(t *testing.T) {
		t.Parallel()
		// A leading minus is arithmetic, not a package qualifier.
		pkg, sym := resolve(t, "-1")
		testkit.Equal(t, pkg, "", "a negative number names no package")
		testkit.Equal(t, sym, "-1", "the number travels verbatim")
	})

	t.Run("passes a raw-quoted literal through untouched", func(t *testing.T) {
		t.Parallel()
		// A backquoted string may hold dots and is still a literal.
		pkg, sym := resolve(t, "`a.b.c`")
		testkit.Equal(t, pkg, "", "a raw literal names no package")
		testkit.Equal(t, sym, "`a.b.c`", "the literal travels verbatim")
	})

	t.Run("reports a qualifier naming no import", func(t *testing.T) {
		t.Parallel()
		// Resolving it to something plausible would emit a reference the file
		// never imports, failing in the consumer's compiler rather than here.
		_, _, err := defaults.Resolve(readerOf(t, fixture(t)), fieldOf(t, fixture(t)), "nowhere.Thing")
		testkit.ErrorIs(t, err, defaults.ErrUnresolvedQualifier, "an unknown qualifier is reported")
	})
}

// resolve splits value against a fixture declaring no imports, which is enough
// for the literal and full-path arms.
func resolve(t *testing.T, value string) (pkg, symbol string) {
	t.Helper()
	s := fixture(t)
	pkg, symbol, err := defaults.Resolve(readerOf(t, s), fieldOf(t, s), value)
	testkit.NoError(t, err, "the value resolves")
	return pkg, symbol
}

// readerOf returns a reader over s.
func readerOf(t *testing.T, s *store.Store) *store.Reader {
	t.Helper()
	return store.NewReader(s)
}

// fieldOf returns the fixture's only field.
func fieldOf(t *testing.T, s *store.Store) *node.Field {
	t.Helper()
	for _, st := range s.Nodes().Structs().Items() {
		for _, f := range st.Fields {
			return f
		}
	}
	t.Fatal("fixture has no field")
	return nil
}

// imported returns a store whose file imports path under alias, optionally
// declaring values as the field's defaults.
func imported(t *testing.T, _, path string, values ...string) *store.Store {
	t.Helper()
	s := storefixture.New().
		Package("cfg", "example.com/cfg").
		Import(path).
		Struct("Config", func(b *storefixture.StructBuilder) {
			b.Pos(position.At("types.go", 1, 1))
			b.Field("Host", storefixture.Named("string"), func(f *storefixture.FieldBuilder) {
				f.Pos(position.At("types.go", 2, 1))
				for _, v := range values {
					f.Directive(storefixture.Directive("default", storefixture.Arg(v)))
				}
			})
		}).
		Build()
	return s
}

// stamped annotates a one-field struct declaring value and returns the stamp.
func stamped(t *testing.T, value string) string {
	t.Helper()
	s := fixture(t, value)
	annotate(t, s)
	return read(t, s)
}

// fixture returns a store whose Config.Host declares each of values in order.
// An empty slice leaves the field undeclared.
func fixture(t *testing.T, values ...string) *store.Store {
	t.Helper()
	return storefixture.New().
		Package("cfg", "example.com/cfg").
		Struct("Config", func(b *storefixture.StructBuilder) {
			b.Field("Host", storefixture.Named("string"), func(f *storefixture.FieldBuilder) {
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
func annotate(t *testing.T, s *store.Store) []diag.Diag {
	t.Helper()
	sink := diag.New()
	ctx := &plugin.AnnotatorContext{Store: s, Reader: store.NewReader(s), Diag: sink}
	if err := defaults.New().Annotate(ctx); err != nil {
		t.Fatalf("Annotate: %v", err)
	}
	return sink.Diagnostics()
}

// read returns the stamp on Config.Host.
func read(t *testing.T, s *store.Store) string {
	t.Helper()
	for _, st := range s.Nodes().Structs().Items() {
		for _, f := range st.Fields {
			if f.Name == "Host" {
				return defaults.Of(f.Meta())
			}
		}
	}
	t.Fatal("fixture has no Config.Host")
	return ""
}
