// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package defaults owns testkit's `//testkit:default` directive: the value a
// generated constructor seeds a field with when the caller supplies none.
//
// # Why its own plugin
//
// The builder generator is the first reader, not the only one. A fixture
// generator seeds the same field the same way, and a model tier needs the
// declared default to state what "unset" means. A directive may be registered
// once per run, so declaring it inside any one generator would make the others
// depend on that generator being registered.
//
// # The literal is carried verbatim
//
// The directive's argument is Go source and is stamped unparsed. `"localhost"`,
// `8080`, `true`, `0.75` and `nil` all reach a template as themselves, which
// costs nothing and avoids a type-directed parser that would have to know every
// literal form Go admits — and would have to be told the field's type to tell
// `0` from `0.0`.
//
// What is checked is that the argument parses as a Go expression. A typo then
// fails here, positioned at the directive, rather than in the consumer's
// compiler against generated code they did not write.
//
// # An explicit zero is not an absent directive
//
// `//testkit:default 0` stamps. A generator reading the stamp sees a value; one
// reading a bare zero cannot tell "seed this to zero" from "no default given",
// and would emit the same constructor either way. That distinction is why the
// stamp is a string rather than a typed value: the empty string is the only
// absence.
package defaults

import (
	"errors"
	"fmt"
	"path"
	"strconv"
	"strings"

	"go.thesmos.sh/eidos/core/directive"
	"go.thesmos.sh/eidos/core/meta"
	"go.thesmos.sh/eidos/node"
	"go.thesmos.sh/eidos/sdk"
	"go.thesmos.sh/eidos/store"
)

// Name is the plugin's identity within the pipeline.
const Name = "defaults"

// Version composes into the pipeline's plugin fingerprint, which frontends
// fold into their cache keys, so a change here invalidates a warm cache
// populated when this annotator stamped differently.
//
// Bump it whenever what gets stamped changes: a different parse, a changed
// validation rule, a new key.
const Version = "1.0.0"

// DirectiveName is the directive this annotator owns, written under testkit's
// namespace as `//testkit:default`.
const DirectiveName sdk.DirectiveName = "default"

// MetaDefault holds the field's default as Go source. Absent, or the empty
// string, means the field declared none — which is distinct from a default of
// zero, and the reason the stamp is not a typed value.
//
//nolint:gochecknoglobals // meta key registration, immutable after init.
var MetaDefault = meta.EnsureKey("testkit.default", meta.StringParser)

// MetaDefaultPkg holds the import path a qualified default resolved to, empty
// for a plain literal. Two keys rather than one string because a rendered file
// has to register the import, which only a reference can carry.
//
//nolint:gochecknoglobals // meta key registration, immutable after init.
var MetaDefaultPkg = meta.EnsureKey("testkit.default.pkg", meta.StringParser)

// Plugin is the defaults annotator.
type Plugin struct{}

// New returns a plugin instance.
func New() *Plugin { return &Plugin{} }

// Name returns [Name].
func (*Plugin) Name() string { return Name }

// Version returns [Version].
func (*Plugin) Version() string { return Version }

// Directives declares the `//testkit:default` schema.
//
// One required positional, because a default with no value is a line the author
// did not finish writing. Negation is denied: a default exists exactly where one
// is declared, so deleting the line is the suppression.
func (*Plugin) Directives() []sdk.DirectiveSchema {
	return []sdk.DirectiveSchema{
		sdk.NewDirective(DirectiveName).
			Describe(
				"Seeds the annotated field with a value when a generated "+
					"constructor is called without one. Takes one positional "+
					"argument, written as the Go literal it will render as — a "+
					"quoted string, a number, a keyword, or nil. Repeating the "+
					"directive takes the last value written.",
			).
			Positional("value", directive.Required()).
			On(node.KindField).
			DenyNegation().
			Build(),
	}
}

// Annotate stamps every field's declared default.
//
// Malformed input is reported and dropped rather than guessed at. A literal
// that does not parse would otherwise reach a template unexamined and emerge as
// a syntax error in generated code, attributed to the generator rather than to
// the line that caused it.
func (*Plugin) Annotate(ctx *sdk.AnnotatorContext) error {
	for _, s := range ctx.Reader.Structs().Slice() {
		for _, f := range s.Fields {
			annotate(ctx, s, f)
		}
	}
	return nil
}

// annotate stamps one field's default.
func annotate(ctx *sdk.AnnotatorContext, s *node.Struct, f *node.Field) {
	for _, dir := range f.Directives() {
		if dir.Name != DirectiveName || len(dir.Args) == 0 {
			continue
		}
		value := dir.Args[0]
		if err := wellFormed(value); err != nil {
			ctx.Diag.Errorf(dir.Pos,
				"%s: default %q on %s.%s: %v",
				Name, value, s.Name, f.Name, err)
			continue
		}
		// Last write wins. A field carrying the directive twice states two
		// intentions; taking the last matches how a reader scans a line list
		// and is what the schema's description promises.
		pkg, symbol, err := Resolve(ctx.Reader, f, value)
		if err != nil {
			ctx.Diag.Errorf(dir.Pos, "%s: default on %s.%s: %v", Name, s.Name, f.Name, err)
			continue
		}
		MetaDefault.Set(f.EnsureMeta(), symbol, Name)
		if pkg != "" {
			MetaDefaultPkg.Set(f.EnsureMeta(), pkg, Name)
		}
	}
}

// ErrMalformedDefault is returned for a value that cannot be Go source.
var ErrMalformedDefault = errors.New("defaults: malformed default")

// wellFormed reports whether value can be Go source.
//
// The check is deliberately shallow. A full expression parse would mean
// importing go/parser, and this tier reads eidos's node model rather than Go
// source — the boundary docs/adr/0003 exists to hold. What it can catch
// without crossing that line is the mistake people actually make: an unbalanced
// quote, which turns the rest of the generated literal into string content and
// produces an error pointing somewhere else entirely.
//
// Anything else passes and is checked by the compiler that reads the generated
// file. A named constant or a conversion cannot be validated here in any case:
// resolving it needs the type information this tier does not have.
func wellFormed(value string) error {
	if strings.TrimSpace(value) == "" {
		return fmt.Errorf("%w: empty", ErrMalformedDefault)
	}
	if value[0] == '"' || value[0] == '`' {
		if _, err := strconv.Unquote(value); err != nil {
			return fmt.Errorf("%w: unbalanced quoting", ErrMalformedDefault)
		}
	}
	return nil
}

// ErrUnresolvedQualifier is returned for a qualifier naming no import.
var ErrUnresolvedQualifier = errors.New("defaults: unresolved qualifier")

// Resolve splits a declared value into the package it names and the symbol
// within it, or reports an empty package for a plain literal.
//
// Two notations, distinguished by splitting on the *last* dot:
//
//	time.Second                     -> resolved against the declaring file's imports
//	example.com/seed.DefaultRegion  -> a full import path, needing no import
//	gopkg.in/yaml.v3.Marshal        -> also a full path; the dots before the
//	                                   last one belong to the path
//
// The second notation exists because an import written only to feed a directive
// is an unused import, which does not compile. Without it, a value can only
// name a package the file already uses for real code.
//
// A literal — quoted, numeric, a keyword — has no qualifier and passes through
// untouched.
func Resolve(r *store.Reader, f *node.Field, value string) (pkg, symbol string, err error) {
	dot := strings.LastIndex(value, ".")
	if dot < 0 || !qualified(value) {
		return "", value, wellFormed(value)
	}
	qualifier, symbol := value[:dot], value[dot+1:]
	if strings.Contains(qualifier, "/") {
		// A full path names itself; nothing to look up.
		return qualifier, symbol, nil
	}
	// The declaring file first, since that is what a reader resolves against.
	// A package-level import is consulted after, because a frontend that
	// records imports per package rather than per file is still describing
	// what this declaration can see.
	for _, file := range r.Files().Slice() {
		if file.Path != f.Pos().File && file.Name != f.Pos().File {
			continue
		}
		if p, ok := lookup(file.Imports, qualifier); ok {
			return p, symbol, nil
		}
	}
	for _, pkg := range r.Packages().Slice() {
		if p, ok := lookup(pkg.Imports, qualifier); ok {
			return p, symbol, nil
		}
	}
	return "", "", fmt.Errorf(
		"%w: %q names no package this file imports; write the full path as "+
			"<import/path>.%s if it is imported only for this directive",
		ErrUnresolvedQualifier, qualifier, symbol,
	)
}

// lookup finds the import a qualifier names, by explicit alias first and by
// the path's last segment otherwise — which is what Go itself does.
func lookup(imports []*node.Import, qualifier string) (string, bool) {
	for _, imp := range imports {
		if imp.Alias == qualifier || path.Base(imp.Path) == qualifier {
			return imp.Path, true
		}
	}
	return "", false
}

// qualified reports whether value reads as a package-qualified symbol rather
// than as a literal. A quoted string or a number can hold a dot without naming
// anything.
func qualified(value string) bool {
	if value == "" || value[0] == '"' || value[0] == '`' {
		return false
	}
	c := value[0]
	return (c < '0' || c > '9') && c != '-' && c != '+'
}

// Package returns the import path a qualified default resolved to, empty when
// the default is a plain literal.
func Package(bag *meta.Bag) string {
	if bag == nil {
		return ""
	}
	out, _ := MetaDefaultPkg.Get(bag)
	return out
}

// Of returns the field's declared default as Go source, or empty when it
// declared none.
func Of(bag *meta.Bag) string {
	if bag == nil {
		return ""
	}
	out, _ := MetaDefault.Get(bag)
	return out
}
