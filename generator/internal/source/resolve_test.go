// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package source_test

import (
	"testing"

	"go.thesmos.sh/eidos/eidostest/storefixture"
	"go.thesmos.sh/eidos/lang/golang"
	"go.thesmos.sh/eidos/sdk"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/generator/internal/source"
)

// A value naming a symbol elsewhere needs an import path, and a qualifier
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
		s := importing(t, "", "example.com/seed")
		pkg, sym, err := source.Resolve(declaringFile(t, s), "seed.Region")
		testkit.NoError(t, err, "an imported qualifier resolves")
		testkit.Equal(t, pkg, "example.com/seed", "the path comes from the import")
		testkit.Equal(t, sym, "Region", "the symbol follows the qualifier")
	})

	t.Run("resolves a qualifier bound by an explicit alias", func(t *testing.T) {
		t.Parallel()
		// The alias is the whole reason a qualifier cannot be derived from the
		// path: nothing about `example.com/gen/shopv1` says `pb`, and a
		// resolver reading the path alone answers with confidence and is wrong.
		s := importing(t, "pb", "example.com/gen/shopv1")
		pkg, sym, err := source.Resolve(declaringFile(t, s), "pb.DefaultTier")
		testkit.NoError(t, err, "an aliased qualifier resolves")
		testkit.Equal(t, pkg, "example.com/gen/shopv1", "the path comes from the aliased import")
		testkit.Equal(t, sym, "DefaultTier", "the symbol follows the alias")
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
		_, _, err := source.Resolve(declaringFile(t, plain(t)), "nowhere.Thing")
		testkit.ErrorIs(t, err, golang.ErrUnresolvedQualifier, "an unknown qualifier is reported")
	})

	t.Run("a qualifier naming no import, with the way out", func(t *testing.T) {
		t.Parallel()
		// An author who reached for a qualifier has no reason to know the
		// full-path notation exists, so the message has to name it.
		_, _, err := source.Resolve(declaringFile(t, plain(t)), "nowhere.Thing")
		testkit.Contains(t, err.Error(), "<import/path>.Thing", "the message names the alternative")
	})

	t.Run("a trailing dot", func(t *testing.T) {
		t.Parallel()
		// A typo, not an exotic input. Accepted, it resolves to an empty
		// symbol — which every reader takes for "nothing declared" — beside a
		// non-empty package, so the file renders `time.` and registers an
		// import nothing uses.
		s := importing(t, "", "time")
		_, _, err := source.Resolve(declaringFile(t, s), "time.")
		testkit.ErrorIs(t, err, golang.ErrBadSymbol, "a trailing dot names no symbol")
	})

	t.Run("a trailing dot on the full-path form", func(t *testing.T) {
		t.Parallel()
		// The same typo one notation over. It reaches a different rule — the
		// last-dot split rather than the first — so refusing it in one place
		// says nothing about the other.
		_, _, err := source.Resolve(declaringFile(t, plain(t)), "example.com/seed.")
		testkit.ErrorIs(t, err, golang.ErrBadSymbol, "a path ending in a dot names no symbol")
	})

	t.Run("a selector chain", func(t *testing.T) {
		t.Parallel()
		// `pkg.a.B` parses as a field access, not as a qualified identifier.
		s := importing(t, "", "example.com/seed")
		_, _, err := source.Resolve(declaringFile(t, s), "seed.a.B")
		testkit.ErrorIs(t, err, golang.ErrBadSymbol, "a chain selects through more than one qualifier")
	})
}

// resolve splits value against a store declaring no imports, which is enough
// for the literal and full-path arms.
func resolve(t *testing.T, value string) (pkg, symbol string) {
	t.Helper()
	pkg, symbol, err := source.Resolve(declaringFile(t, plain(t)), value)
	testkit.NoError(t, err, "the value resolves")
	return pkg, symbol
}

// declaringFile returns the file the store positioned its struct in, which is
// where a qualifier resolves.
func declaringFile(t *testing.T, s *sdk.Store) *sdk.File {
	t.Helper()
	pkg, _ := s.Nodes().Packages().ByQName(resolvePkg)
	for _, st := range s.Nodes().Structs().Items() {
		return golang.FileOf(pkg, st)
	}
	t.Fatal("store has no struct")
	return nil
}

// resolvePkg is the import path these stores declare their struct in.
const resolvePkg = "example.com/cfg"

// resolveFile is the file they position it in. Named explicitly rather than
// left to the builder's synthesised position, because a qualifier resolves
// against a file's import block and a declaration whose position names no
// declared file has no imports in scope.
const resolveFile = "types.go"

// plain returns a store whose file imports nothing, which is what the literal
// and full-path arms resolve against.
func plain(t *testing.T) *sdk.Store {
	t.Helper()
	return declaring(t, nil)
}

// importing returns a store whose file imports path, under alias when one is
// given.
func importing(t *testing.T, alias, path string) *sdk.Store {
	t.Helper()
	return declaring(t, func(f *storefixture.FileBuilder) { f.ImportAs(alias, path) })
}

// declaring builds a one-field struct positioned in a file the caller sets up.
func declaring(t *testing.T, file func(*storefixture.FileBuilder)) *sdk.Store {
	t.Helper()
	return storefixture.New().
		Package("cfg", resolvePkg).
		File(resolveFile, file).
		Struct("Config", func(b *storefixture.StructBuilder) {
			b.Pos(sdk.At(resolveFile, 1, 1))
			b.Field("Host", storefixture.Named("string"), func(f *storefixture.FieldBuilder) {
				f.Pos(sdk.At(resolveFile, 2, 1))
			})
		}).
		Build()
}
