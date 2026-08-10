// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package defaults_test

import (
	"testing"

	"go.thesmos.sh/eidos/eidostest/plugintest"
	"go.thesmos.sh/eidos/eidostest/storefixture"
	"go.thesmos.sh/eidos/lang/golang"
	"go.thesmos.sh/eidos/sdk"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/generator/defaults"
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

	t.Run("passes a leading-dot float through untouched", func(t *testing.T) {
		t.Parallel()
		// `.5` is a legal Go float literal. Read as a qualifier it splits into
		// an empty qualifier and the symbol `5`, and an empty qualifier matches
		// every un-aliased import — so the first import in the file wins and
		// the generated builder says `http.5`.
		pkg, sym := resolve(t, ".5")
		testkit.Equal(t, pkg, "", "a leading dot is a decimal point, not a selector")
		testkit.Equal(t, sym, ".5", "the number travels verbatim")
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
		s := imported(t, "", "example.com/seed")
		pkg, sym, err := defaults.Resolve(fileOf(t, s), "seed.Region")
		testkit.NoError(t, err, "an imported qualifier resolves")
		testkit.Equal(t, pkg, "example.com/seed", "the path comes from the import")
		testkit.Equal(t, sym, "Region", "the symbol follows the qualifier")
	})

	t.Run("resolves a qualifier bound by an explicit alias", func(t *testing.T) {
		t.Parallel()
		// The alias is the whole reason a qualifier cannot be derived from the
		// path: nothing about `example.com/gen/shopv1` says `pb`, and a
		// resolver reading the path alone answers with confidence and is wrong.
		s := imported(t, "pb", "example.com/gen/shopv1")
		pkg, sym, err := defaults.Resolve(fileOf(t, s), "pb.DefaultTier")
		testkit.NoError(t, err, "an aliased qualifier resolves")
		testkit.Equal(t, pkg, "example.com/gen/shopv1", "the path comes from the aliased import")
		testkit.Equal(t, sym, "DefaultTier", "the symbol follows the alias")
	})

	t.Run("stamps the resolved package alongside the symbol", func(t *testing.T) {
		t.Parallel()
		// A rendered file has to register the import, which only a reference
		// carries — so the path travels beside the symbol rather than inside it.
		s := imported(t, "", "example.com/seed", "seed.Region")
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
		// bag at all — and both accessors are called on every field. What this
		// pins is eidos's own contract: a nil meta.Bag reads as empty, so
		// neither accessor needs a guard of its own.
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

	t.Run("passes a bare identifier through untouched", func(t *testing.T) {
		t.Parallel()
		// A constant the declaring package owns renders as itself and needs no
		// import; qualifying it against the source package would emit a
		// self-import.
		pkg, sym := resolve(t, "MaxRetries")
		testkit.Equal(t, pkg, "", "an in-package constant names no package")
		testkit.Equal(t, sym, "MaxRetries", "the identifier travels verbatim")
	})
}

// A value that is neither a literal nor a resolvable symbol is refused at the
// directive. Resolving it to something plausible emits a reference the file
// never imports, failing in the consumer's compiler against generated code.
func TestResolveRejects(t *testing.T) {
	t.Parallel()

	t.Run("a qualifier naming no import", func(t *testing.T) {
		t.Parallel()
		_, _, err := defaults.Resolve(fileOf(t, fixture(t)), "nowhere.Thing")
		testkit.ErrorIs(t, err, golang.ErrUnresolvedQualifier, "an unknown qualifier is reported")
	})

	t.Run("a qualifier naming no import, with the way out", func(t *testing.T) {
		t.Parallel()
		// An author who reached for a qualifier has no reason to know the
		// full-path notation exists, so the message has to name it.
		_, _, err := defaults.Resolve(fileOf(t, fixture(t)), "nowhere.Thing")
		testkit.Contains(t, err.Error(), "<import/path>.Thing", "the message names the alternative")
	})

	t.Run("a trailing dot", func(t *testing.T) {
		t.Parallel()
		// A typo, not an exotic input. Accepted, it stamps an empty symbol —
		// which every reader takes for "no default declared" — beside a
		// non-empty package, so the file renders `time.` and registers an
		// import nothing uses.
		s := imported(t, "", "time")
		_, _, err := defaults.Resolve(fileOf(t, s), "time.")
		testkit.ErrorIs(t, err, golang.ErrBadSymbol, "a trailing dot names no symbol")
	})

	t.Run("a trailing dot on the full-path form", func(t *testing.T) {
		t.Parallel()
		// The same typo one notation over. It reaches a different rule — the
		// last-dot split rather than the first — so refusing it in one place
		// says nothing about the other.
		_, _, err := defaults.Resolve(fileOf(t, fixture(t)), "example.com/seed.")
		testkit.ErrorIs(t, err, golang.ErrBadSymbol, "a path ending in a dot names no symbol")
	})

	t.Run("a selector chain", func(t *testing.T) {
		t.Parallel()
		// `pkg.a.B` parses as a field access, not as a qualified identifier.
		s := imported(t, "", "example.com/seed")
		_, _, err := defaults.Resolve(fileOf(t, s), "seed.a.B")
		testkit.ErrorIs(t, err, golang.ErrBadSymbol, "a chain selects through more than one qualifier")
	})
}

// resolve splits value against a fixture declaring no imports, which is enough
// for the literal and full-path arms.
func resolve(t *testing.T, value string) (pkg, symbol string) {
	t.Helper()
	s := fixture(t)
	pkg, symbol, err := defaults.Resolve(fileOf(t, s), value)
	testkit.NoError(t, err, "the value resolves")
	return pkg, symbol
}

// fileOf returns the file the fixture positioned its struct in, which is where
// a qualifier resolves. Nil when the fixture declared none.
func fileOf(t *testing.T, s *sdk.Store) *sdk.File {
	t.Helper()
	pkg, _ := s.Nodes().Packages().ByQName(fixturePkg)
	for _, st := range s.Nodes().Structs().Items() {
		return golang.FileOf(pkg, st)
	}
	t.Fatal("fixture has no struct")
	return nil
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
				return defaults.Of(f.Meta())
			}
		}
	}
	t.Fatal("fixture has no Config.Host")
	return ""
}
