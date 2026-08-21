// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package defaults

import (
	"go.thesmos.sh/eidos/lang/golang"
	"go.thesmos.sh/eidos/sdk"
	sdkgolang "go.thesmos.sh/eidos/sdk/golang"

	"go.thesmos.sh/testkit/generator/internal/source"
	"go.thesmos.sh/testkit/generator/internal/stamp"
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
	pkg, symbol, err := source.Resolve(file, dir.Args[0])
	if err != nil {
		// The value itself comes from the error: [source.Resolve] quotes it, because
		// its other caller reads a directive key rather than a positional and
		// has nothing else to name.
		ctx.Diag.Errorf(dir.Pos, "%s: default on %s.%s: %v", Name, owner, name, err)
		return
	}
	stamp.MetaDefault.Set(bag, symbol, Name)
	if pkg != "" {
		stamp.MetaDefaultPkg.Set(bag, pkg, Name)
	}
}
