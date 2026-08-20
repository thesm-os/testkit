// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package defaults

import (
	"errors"
	"fmt"
	"strings"

	"go.thesmos.sh/eidos/lang/golang"
	"go.thesmos.sh/eidos/sdk"
	sdkgolang "go.thesmos.sh/eidos/sdk/golang"
)

// Name is the plugin's identity within the pipeline.
const Name = "defaults"

// Version composes into the pipeline's plugin fingerprint, which frontends
// fold into their cache keys, so a change here invalidates a warm cache
// populated when this annotator stamped differently.
//
// Bump it whenever what gets stamped changes: a different parse, a changed
// validation rule, a new key.
const Version = "1.1.0"

// DirectiveName is the directive this annotator owns, written under testkit's
// namespace as `//testkit:default`.
const DirectiveName sdk.DirectiveName = "default"

// MetaDefault holds the field's default as Go source. Absent, or the empty
// string, means the field declared none — which is distinct from a default of
// zero, and the reason the stamp is not a typed value.
//
//nolint:gochecknoglobals // meta key registration, immutable after init.
var MetaDefault = sdk.EnsureKey("testkit.default", sdk.StringParser)

// MetaDefaultPkg holds the import path a qualified default resolved to, empty
// for a plain literal. Two keys rather than one string because a rendered file
// has to register the import, which only a reference can carry.
//
//nolint:gochecknoglobals // meta key registration, immutable after init.
var MetaDefaultPkg = sdk.EnsureKey("testkit.default.pkg", sdk.StringParser)

// Plugin is the defaults annotator. The zero value is unusable; go through
// [New], which builds the embedded base.
type Plugin struct{ *sdkgolang.Base }

// New returns a plugin instance.
//
// [sdkgolang.NewPlugin] rather than NewGenerator: this plugin ships no template
// tree and emits no file. It stamps metadata a later generator reads, and
// declaring an output it never writes would give Layout a filename to compose
// for a contribution that never arrives.
func New() *Plugin {
	return &Plugin{Base: sdkgolang.NewPlugin(Name).
		Version(Version).
		Directives(directives()...).
		Build()}
}

// directives declares the `//testkit:default` schema.
//
// One required positional, because a default with no value is a line the author
// did not finish writing. Negation is denied: a default exists exactly where one
// is declared, so deleting the line is the suppression.
func directives() []sdk.DirectiveSchema {
	return []sdk.DirectiveSchema{
		sdk.NewDirective(DirectiveName).
			Describe(
				"Seeds the annotated field with a value when a generated "+
					"constructor is called without one. Takes one positional "+
					"argument, written as the Go literal it will render as — a "+
					"quoted string, a number, a keyword, or nil. Repeating the "+
					"directive takes the last value written.",
			).
			Positional("value", sdk.Required()).
			On(sdk.NodeKindField, sdk.NodeKindAlias).
			// One positional and nothing else: the value is the whole
			// directive, and a key beside it names something this generator
			// has never read.
			DenyKeys().
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
		// Resolved once per struct rather than per field: every field of a
		// struct is declared in the struct's own file, so the answer cannot
		// differ between them.
		file := fileOf(ctx.Reader, s)
		for _, f := range s.Fields {
			annotate(ctx, file, s.Name, f.Name, f.Directives(), f.EnsureMeta())
		}
	}
	// The type-level arm, which pairs with the role annotator's: a bare
	// parameter has no field to stamp, so both the role and the value it
	// draws are declared on the named type instead.
	for _, a := range ctx.Reader.Aliases().Slice() {
		pkg, _ := ctx.Reader.PackageAt(a.Package)
		annotate(ctx, golang.FileOf(pkg, a), a.Name, a.Name, a.Directives(), a.EnsureMeta())
	}
	return nil
}

// fileOf returns the source file that declared s, or nil when the run recorded
// none.
//
// Go scopes a qualifier to the file whose import block bound it — `pb.Event`
// means whatever `pb` was aliased to *there* and means nothing one file over —
// so resolving one needs the file, not the package. [golang.FileOf] is the step
// from a declaration to it: a position carries a path while a package keys its
// files by basename, and a lookup composed at the call site is one `path.Base`
// from always missing, silently.
//
// Nil for a positionless struct, which is the honest answer: a synthetic
// declaration has no file and therefore no imports in scope.
func fileOf(r *sdk.StoreReader, s *sdk.Struct) *sdk.File {
	pkg, _ := r.PackageAt(s.Package)
	return golang.FileOf(pkg, s)
}

// annotate stamps one field's default.
//
// Last write wins. A field carrying the directive twice states two intentions;
// taking the last matches how a reader scans a line list and is what the
// schema's description promises. [sdk.Field.Directive] answers "is this
// declared" and is first-wins, which is the opposite rule — hence
// [sdk.Last] rather than the method.
func annotate(
	ctx *sdk.AnnotatorContext,
	file *sdk.File,
	owner, name string,
	directives []*sdk.Directive,
	bag *sdk.Bag,
) {
	dir := sdk.Last(directives, DirectiveName)
	if dir == nil || len(dir.Args) == 0 {
		return
	}
	pkg, symbol, err := Resolve(file, dir.Args[0])
	if err != nil {
		// The value itself comes from the error: [Resolve] quotes it, because
		// its other caller reads a directive key rather than a positional and
		// has nothing else to name.
		ctx.Diag.Errorf(dir.Pos, "%s: default on %s.%s: %v", Name, owner, name, err)
		return
	}
	MetaDefault.Set(bag, symbol, Name)
	if pkg != "" {
		MetaDefaultPkg.Set(bag, pkg, Name)
	}
}

// Resolve splits a declared value into the package it names and the symbol
// within it, or reports an empty package for a plain literal.
//
// Two notations, told apart by whether the qualifier holds a slash:
//
//	time.Second                     -> resolved against file's import block
//	example.com/seed.DefaultRegion  -> a full import path, needing no import
//	gopkg.in/yaml.v3.Marshal        -> also a full path; the dots before the
//	                                   last one belong to the path
//
// The second notation exists because an import written only to feed a directive
// is an unused import, which does not compile. Without it, a value can only
// name a package the file already uses for real code.
//
// A literal — quoted, numeric, a keyword — has no qualifier and passes through
// untouched, checked only for the quoting fault that would swallow the rest of
// the generated line.
//
// A nil file resolves no qualifier at all. That is what a positionless
// declaration gets, and it reports [golang.ErrUnresolvedQualifier] rather than
// guessing against some other file's imports — the shape that made this
// function panic when a caller had no declaration to hand.
func Resolve(file *sdk.File, value string) (pkg, symbol string, err error) {
	if malformed := golang.IsWellFormedLiteral(value); malformed != nil {
		return "", "", fmt.Errorf("%q: %w", value, malformed)
	}
	if !qualified(value) {
		return "", value, nil
	}
	ref, err := resolveRef(file, value)
	if err != nil {
		return "", "", err
	}
	ext, ok := ref.(*sdk.ExternalRef)
	if !ok {
		// A bare identifier — a constant the declaring package owns. It renders
		// as itself and registers no import, which is what an empty source
		// package asks [golang.RefFor] for: the stamp is read by whichever file
		// later renders it, and that file is not known here.
		return "", value, nil
	}
	return ext.Package, ext.Name, nil
}

// resolveRef hands value to whichever of eidos's two rules its notation calls
// for.
//
// The two split on opposite dots, and that is the whole distinction. An import
// path may hold dots, so the full-path form splits from the right; a Go
// qualifier is one identifier and cannot hold one, so the source form splits
// from the left. Reading source text with the right-hand rule manufactures a
// qualifier that is not an identifier.
//
// A slash before the last dot is what picks the first: no Go qualifier holds
// one, and every import path worth writing in a directive does.
func resolveRef(file *sdk.File, value string) (sdk.Ref, error) {
	if dot := strings.LastIndex(value, "."); dot > 0 && strings.Contains(value[:dot], "/") {
		ref, err := golang.RefForQualified(value, "")
		if err != nil {
			return nil, fmt.Errorf("%q: %w", value, err)
		}
		return ref, nil
	}
	ref, err := golang.ResolveQualified(file, value, "")
	switch {
	case errors.Is(err, golang.ErrUnresolvedQualifier):
		// The full-path form is the way out of this, and an author who reached
		// for a qualifier has no reason to know it exists.
		_, symbol := golang.QualifierOf(value)
		return nil, fmt.Errorf(
			"%q: %w; write the full path as <import/path>.%s if the package is "+
				"imported only for this directive", value, err, symbol,
		)
	case err != nil:
		return nil, fmt.Errorf("%q: %w", value, err)
	}
	return ref, nil
}

// qualified reports whether value reads as a package-qualified symbol rather
// than as a literal. A quoted string or a number can hold a dot without naming
// anything.
//
// A leading dot is a number too: `.5` is a legal Go float literal and Go's own
// scanner reads it as one. Reading it as a qualifier splits it into an empty
// qualifier and the symbol `5`, which matches every un-aliased import — so the
// first import in the file wins and the generated builder says `http.5`.
func qualified(value string) bool {
	if value == "" || value[0] == '"' || value[0] == '`' {
		return false
	}
	c := value[0]
	return (c < '0' || c > '9') && c != '-' && c != '+' && c != '.'
}

// Package returns the import path a qualified default resolved to, empty when
// the default is a plain literal.
func Package(bag *sdk.Bag) string {
	out, _ := MetaDefaultPkg.Get(bag)
	return out
}

// Of returns the field's declared default as Go source, or empty when it
// declared none.
func Of(bag *sdk.Bag) string {
	out, _ := MetaDefault.Get(bag)
	return out
}
