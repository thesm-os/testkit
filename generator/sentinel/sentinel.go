// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package sentinel

import (
	"fmt"
	"io/fs"
	"path"
	"sort"
	"strings"
	"text/template"

	"go.thesmos.sh/eidos/emit"
	"go.thesmos.sh/eidos/lang/golang"
	"go.thesmos.sh/eidos/node"
	"go.thesmos.sh/eidos/sdk"

	"go.thesmos.sh/testkit/generator/internal/nodes"
	"go.thesmos.sh/testkit/generator/internal/samples"
)

// Name is the plugin's stable identifier.
const Name = "sentinel"

// Capability is the label the plugin advertises so a downstream consumer can
// declare a documentary dependency on sentinel checks.
const Capability = "sentinel"

// DirectiveName is the bare directive name — without the `//testkit:` prefix —
// that opts a package in.
const DirectiveName sdk.DirectiveName = "sentinel"

// NoOverlapName is the directive naming another package this one's sentinels
// must stay distinct from.
//
// A separate directive rather than a key on [DirectiveName] because it repeats:
// a package may declare several neighbours, and each line unions into one set.
// A key would have to encode the list into one value.
const NoOverlapName sdk.DirectiveName = "sentinel-no-overlap-with"

// PrefixKey overrides the prefix every sentinel's message must begin with, and
// [PrefixOff] suppresses the check.
const (
	PrefixKey = "prefix"
	PrefixOff = "off"
)

// ErrPrefix is the identifier prefix a package-level variable needs to be read
// as a sentinel.
//
// Convention rather than declaration: this is the name Go codebases already
// use, so a package opts its errors in by being written the ordinary way. The
// cost is that a sentinel named otherwise is not found — which is why the
// generated file lists what it covers, so an absence is visible.
const ErrPrefix = "Err"

// ErrorMethod is the method an exported type must declare to be read as a
// custom error, and the two optional ones that earn it further checks.
const (
	ErrorMethod  = "Error"
	IsMethod     = "Is"
	UnwrapMethod = "Unwrap"
)

// SlotName is the [emit.File] slot the checks land in.
const SlotName = "top"

// KindTests is the plugin-defined emit kind. The backend resolves a template by
// the kind's string value, so the constant doubles as the template's name.
const KindTests sdk.Kind = "sentinel.tests"

// Version composes into the pipeline's plugin fingerprint. Bump it on any
// change to what this plugin emits — the projection or the template alike.
const Version = "1.0.0"

// langGo is the backend language identifier the per-language adapters key on.
const langGo = golang.Language

// Plugin is the sentinel check generator.
type Plugin struct{}

// New returns a fresh plugin instance.
func New() *Plugin { return &Plugin{} }

// Name returns [Name].
func (*Plugin) Name() string { return Name }

// Version returns [Version].
func (*Plugin) Version() string { return Version }

// Priority places the plugin in the foundation bucket: it reads source and
// contributes to no other plugin's output.
func (*Plugin) Priority() sdk.Priority { return sdk.GeneratorFoundation }

// Provides advertises [Capability].
func (*Plugin) Provides() []string { return []string{Capability} }

// Requires returns nil — the plugin reads source declarations and depends on
// no other plugin's contribution.
func (*Plugin) Requires() []string { return nil }

// Directives declares both schemas.
func (*Plugin) Directives() []sdk.DirectiveSchema {
	return []sdk.DirectiveSchema{
		sdk.NewDirective(DirectiveName).
			Describe(
				"Generates checks over the host package's error contract: every " +
					"Err* variable's message prefix, uniqueness and mutual " +
					"distinctness, survival under wrapping, and one check per " +
					"exported type declaring Error. `prefix=<value>` overrides the " +
					"expected message prefix; `prefix=off` suppresses that check. " +
					"The negated form is rejected — removing the directive is the " +
					"suppression.",
			).
			AllowedKeys(PrefixKey).
			On(node.KindPackage).
			DenyNegation().
			Build(),
		sdk.NewDirective(NoOverlapName).
			Describe(
				"Names another package whose sentinels must not satisfy errors.Is " +
					"against this package's. Takes one import path and repeats; " +
					"each line adds to the set.",
			).
			Positional("package").
			On(node.KindPackage).
			DenyNegation().
			Build(),
	}
}

// Outputs dispatches to the per-language adapter.
func (*Plugin) Outputs(lang string) []sdk.Output {
	if lang == langGo {
		return GoOutputs()
	}
	return nil
}

// Templates dispatches to the per-language adapter's template tree.
func (*Plugin) Templates(lang string) (fs.FS, bool) {
	if lang == langGo {
		return GoTemplates()
	}
	return nil, false
}

// TemplateFuncs dispatches to the per-language adapter's funcmap.
func (*Plugin) TemplateFuncs(lang string) template.FuncMap {
	if lang == langGo {
		return GoFuncMap()
	}
	return nil
}

// TemplateOverrides returns nil — the plugin replaces no canonical funcmap
// entry.
func (*Plugin) TemplateOverrides(string) template.FuncMap { return nil }

// Sentinel is one package-level error variable.
type Sentinel struct {
	// Name is the identifier, which the checks report by so a failure names
	// the declaration rather than its message.
	Name string

	// Ref qualifies the variable. The checks live in the package's external
	// test package, so nothing is reachable unqualified.
	Ref *emit.Expr
}

// Field is one exported field of a custom error type, with a value the check
// writes into it.
type Field struct {
	Name string

	// Sample is the value as source text, empty when the field is not checked.
	//
	// String fields only. A message renders a number through a format verb
	// whose width and base the generator cannot see, so asserting that "42"
	// appears in the output would fail against `%03d` for a field that is
	// perfectly well reported. A string arrives verbatim or not at all, which
	// is the case where containment answers the question asked.
	Sample string
}

// ErrType is one exported type declaring Error.
type ErrType struct {
	Name string
	Ref  *emit.Expr

	// Pointer reports whether Error is declared on the pointer receiver, which
	// decides whether a check writes `&T{}` or `T{}`.
	Pointer bool

	// CauseField is the exported field holding the wrapped error, empty when
	// the type declares none. Without one there is no cause to hand the type,
	// so its unwrap check is not emitted rather than emitted against a nil.
	CauseField string

	// HasIs and HasUnwrap record the optional methods. Each earns a check, and
	// a type declaring neither gets neither rather than a vacuous one.
	HasIs, HasUnwrap bool

	// Fields are the exported fields a message is expected to carry.
	Fields []Field
}

// Checked reports whether any field carries a value a check can write, which
// is what decides whether the message-completeness check is emitted.
func (e ErrType) Checked() bool {
	for _, f := range e.Fields {
		if f.Sample != "" {
			return true
		}
	}
	return false
}

// Neighbour is another package this one's sentinels must stay distinct from.
type Neighbour struct {
	// Path is the import path as written in the directive, and Name its last
	// segment — which is what a subtest name reads better as.
	Path, Name string

	Sentinels []Sentinel
}

// Tests is the emit value rendered into the output.
type Tests struct {
	sdk.BaseEmit

	// PackageName is the source package's identifier, which names the check.
	PackageName string

	// Prefix is what every sentinel's message must begin with, empty when the
	// check is suppressed.
	Prefix string

	Sentinels  []Sentinel
	ErrTypes   []ErrType
	Neighbours []Neighbour
}

// Kind returns [KindTests].
func (*Tests) Kind() sdk.Kind { return KindTests }

// Generate walks every package carrying `//testkit:sentinel` and queues one
// [Tests] against it.
//
// A package with neither a sentinel nor an error type is reported: the
// directive says its errors are a contract, and a file asserting nothing about
// an empty set would read as though they had been checked.
func (*Plugin) Generate(ctx *sdk.GeneratorContext) error {
	c := sdk.NewProvenance(Name, sdk.EmitTarget{})
	byPackage := sentinelsByPackage(ctx)

	for _, pkg := range ctx.Reader.Packages().Slice() {
		if !pkg.HasPositiveDirective(DirectiveName) {
			continue
		}
		found := byPackage[pkg.Path]
		types := errTypesOf(ctx, pkg.Path)
		if len(found) == 0 && len(types) == 0 {
			ctx.Diag.Errorf(pkg.Pos(),
				"%s: package %q carries //testkit:%s but declares no %s* variable and no type with an %s method",
				Name, pkg.Path, DirectiveName, ErrPrefix, ErrorMethod)
			continue
		}
		anchor := anchorOf(ctx, pkg.Path, found)
		if anchor == nil {
			continue
		}
		value := &Tests{
			BaseEmit:    sdk.BaseEmit{OriginNode: anchor, SetByName: c.SetBy(), SourcePos: anchor.Pos()},
			PackageName: pkg.Name,
			Prefix:      prefixOf(pkg),
			Sentinels:   found,
			ErrTypes:    types,
			Neighbours:  neighboursOf(ctx, pkg, byPackage),
		}
		prov := c.Provenance(string(KindTests) + "." + pkg.Name)
		if err := ctx.Store.Emit().AppendOriginSlot(anchor, SlotName, value, prov); err != nil {
			return fmt.Errorf("%s: append %s slot for %q: %w", Name, KindTests, pkg.Path, err)
		}
	}
	return nil
}

// sentinelsByPackage indexes every package's sentinel set.
//
// Built once for the whole run rather than per package, because the
// cross-package check needs a neighbour's set and the neighbour may be
// annotated, unannotated, or generated later — reading it from the same index
// keeps one answer to "what are that package's sentinels".
func sentinelsByPackage(ctx *sdk.GeneratorContext) map[string][]Sentinel {
	out := map[string][]Sentinel{}
	for _, v := range ctx.Reader.Variables().Slice() {
		if !strings.HasPrefix(v.Name, ErrPrefix) || !golang.IsExported(v.Name) {
			continue
		}
		out[v.Package] = append(out[v.Package], Sentinel{
			Name: v.Name,
			Ref:  sdk.NewExternal(v.Package, v.Name),
		})
	}
	return out
}

// anchorOf returns the node the output file is composed from.
//
// Layout builds the filename from the origin's source basename, so the anchor
// decides where the checks land. The first sentinel in source order, or failing
// that the first error type, puts them beside the declarations they are about.
func anchorOf(ctx *sdk.GeneratorContext, pkg string, found []Sentinel) node.Node {
	if len(found) > 0 {
		for _, v := range ctx.Reader.Variables().Slice() {
			if v.Package == pkg && v.Name == found[0].Name {
				return v
			}
		}
	}
	for _, s := range ctx.Reader.Structs().Slice() {
		if s.Package == pkg && nodes.Declares(s, ErrorMethod) {
			return s
		}
	}
	return nil
}

// prefixOf resolves what every sentinel's message must begin with, or empty
// when the check is suppressed.
func prefixOf(pkg *node.Package) string {
	for _, dir := range pkg.Directives() {
		if string(dir.Name) != string(DirectiveName) {
			continue
		}
		raw, declared := dir.KV[PrefixKey]
		switch {
		case !declared:
			continue
		case raw == PrefixOff || raw == "":
			return ""
		default:
			return raw + ": "
		}
	}
	return pkg.Name + ": "
}

// neighboursOf resolves every package named by a no-overlap directive.
//
// A neighbour naming no sentinels is kept with an empty set rather than
// dropped: the rendered file lists it, so a directive pointing at a package
// that has none is visible as an empty list instead of as a missing check.
func neighboursOf(ctx *sdk.GeneratorContext, pkg *node.Package, byPackage map[string][]Sentinel) []Neighbour {
	var out []Neighbour
	for _, dir := range pkg.Directives() {
		if string(dir.Name) != string(NoOverlapName) || len(dir.Args) == 0 {
			continue
		}
		p := dir.Args[0]
		if p == pkg.Path {
			ctx.Diag.Errorf(pkg.Pos(),
				"%s: package %q declares //testkit:%s against itself",
				Name, pkg.Path, NoOverlapName)
			continue
		}
		out = append(out, Neighbour{Path: p, Name: path.Base(p), Sentinels: byPackage[p]})
	}
	return out
}

// errTypesOf lifts every exported type in pkg that declares Error.
func errTypesOf(ctx *sdk.GeneratorContext, pkg string) []ErrType {
	var out []ErrType
	for _, s := range ctx.Reader.Structs().Slice() {
		if s.Package != pkg || !golang.IsExported(s.Name) || !nodes.Declares(s, ErrorMethod) {
			continue
		}
		out = append(out, ErrType{
			Name:       s.Name,
			Ref:        sdk.NewExternal(s.Package, s.Name),
			Pointer:    nodes.PointerReceiver(s, ErrorMethod),
			HasIs:      nodes.Declares(s, IsMethod),
			HasUnwrap:  nodes.Declares(s, UnwrapMethod),
			CauseField: nodes.FieldOfType(s, "error"),
			Fields:     fieldsOf(s),
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

// fieldsOf lifts the exported fields whose values a message is expected to
// carry.
func fieldsOf(s *node.Struct) []Field {
	out := make([]Field, 0, len(s.Fields))
	for _, f := range s.Fields {
		if !golang.IsExported(f.Name) || f.Type == nil {
			continue
		}
		sample := ""
		if f.Type.IsBuiltin() && f.Type.Name == "string" {
			sample, _ = samples.For(f.Type.Name, f.Name)
		}
		out = append(out, Field{Name: f.Name, Sample: sample})
	}
	return out
}
