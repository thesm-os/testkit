// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package sentinel

import (
	"fmt"
	"sort"

	"go.thesmos.sh/eidos/lang/golang"
	"go.thesmos.sh/eidos/sdk"
	sdkgolang "go.thesmos.sh/eidos/sdk/golang"
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
//
// It is the prefix [golang.SentinelName] composes with. A generator that
// emitted a sentinel under eidos's convention and a detector that missed it
// would be the same rule written twice, so sentinel_test.go pins the two
// together.
const ErrPrefix = "Err"

// ErrorMethod is the method an exported type must declare to be read as a
// custom error, and the two optional ones that earn it further checks.
const (
	ErrorMethod  = "Error"
	IsMethod     = "Is"
	UnwrapMethod = "Unwrap"
)

// SlotName is the [sdk.EmitFile] slot the checks land in.
const SlotName = "top"

// KindTests is the plugin-defined emit kind. The backend resolves a template by
// the kind's string value, so the constant doubles as the template's name.
const KindTests sdk.Kind = "sentinel.tests"

// Version composes into the pipeline's plugin fingerprint. Bump it on any
// change to what this plugin emits — the projection or the template alike.
const Version = "1.0.0"

// Plugin is the sentinel check generator.
//
// The embedded [sdkgolang.Base] answers every declaration the pipeline asks
// for — name, version, priority, capabilities, directives, outputs, templates
// and the template funcmap — so the only method this package writes is
// [Plugin.Generate].
type Plugin struct{ *sdkgolang.Base }

// New returns a fresh plugin instance.
//
// # Failure mode
//
// [sdkgolang.Builder.Build] panics on a declaration the pipeline cannot serve —
// a missing template tree, a suffix-less output. Every such mistake is in this
// function rather than in a run's input, so it fires on the first construction
// in any test instead of rendering a short file and failing nowhere.
func New() *Plugin {
	return &Plugin{Base: sdkgolang.NewGenerator(Name, goTemplatesFS, GoOutputs()...).
		Version(Version).
		// The foundation bucket: the plugin reads source and contributes to no
		// other plugin's output. Nothing is required for the same reason — it
		// depends on no other plugin's contribution, only on declarations the
		// frontend already loaded.
		Priority(sdk.GeneratorFoundation).
		Provides(Capability).
		Directives(directives()...).
		// Registered as `ref`, which the builder namespaces to `sentinel_ref`
		// exactly as it does the shared bundle's entries: the backend rejects
		// two plugins registering one extension name, so an unprefixed helper
		// would fail every run rather than one output.
		Build()}
}

// directives declares both schemas.
func directives() []sdk.DirectiveSchema {
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
			On(sdk.NodeKindPackage).
			DenyNegation().
			Build(),
		sdk.NewDirective(NoOverlapName).
			Describe(
				"Names another package whose sentinels must not satisfy errors.Is " +
					"against this package's. Takes one import path and repeats; " +
					"each line adds to the set.",
			).
			Positional("package").
			On(sdk.NodeKindPackage).
			DenyNegation().
			Build(),
	}
}

// Sentinel is one package-level error variable.
type Sentinel struct {
	// Name is the identifier, which the checks report by so a failure names
	// the declaration rather than its message.
	Name string

	// Ref qualifies the variable. The checks live in the package's external
	// test package, so nothing is reachable unqualified.
	Ref *sdk.Expr
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
	Ref  *sdk.Expr

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

	// decl is the declaration this was projected from, carried so the anchor
	// comes from what errTypesOf already found rather than from a second scan
	// that could disagree with it.
	decl *sdk.Struct
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
	// Path is the import path as written in the directive, and Name the package
	// clause it implies — which is what a subtest name reads better as.
	//
	// The clause rather than the path's last element: `example.com/other/v2` is
	// package `other`, and a subtest named for `v2` names nothing the reader
	// can find.
	Path, Name string

	Sentinels []Sentinel
}

// Tests is the emit value rendered into the output.
type Tests struct {
	sdk.BaseEmit
	RuntimePaths

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
	c := sdk.NewProvenance(Name)
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
		anchor := anchorOf(ctx, pkg.Path, found, types)
		value := &Tests{
			BaseEmit:     sdk.EmitBase(c, anchor),
			RuntimePaths: GoRuntime(),
			PackageName:  pkg.Name,
			Prefix:       prefixOf(pkg),
			Sentinels:    found,
			ErrTypes:     types,
			Neighbours:   neighboursOf(ctx, pkg, byPackage),
		}
		// Identified by the package rather than by the anchor: the anchor is
		// whichever declaration the package happened to offer, and naming it
		// would move this value's identifier when an unrelated type is renamed.
		if err := sdk.QueueEmitAs(ctx.Store.Emit(), c, SlotName, anchor, pkg.Name, value); err != nil {
			// Wrapped even though the queue names the plugin and the slot: what
			// it cannot name is which declaration the run was on when it failed,
			// and that is the only part a reader needs to find the source line.
			return fmt.Errorf("%s: queue package %q: %w", Name, pkg.Path, err)
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
		// One question, not two. A name beginning `Err` is exported by that
		// fact alone, so the second test could never fail and was reporting
		// nothing about any input.
		if !golang.IsSentinelName(v.Name) {
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
// that the error type [errTypesOf] found first, puts them beside the
// declarations they are about.
//
// The error type comes from what that function already resolved rather than
// from a second scan. The two asked different questions once — one read the
// promoted method set, the other only the declarations — so a package whose
// only error type inherits its contract was found to have one and then anchored
// nowhere, and its checks were dropped without a diagnostic.
//
// One of the two is always available: the caller refuses a package declaring
// neither before it gets here.
func anchorOf(ctx *sdk.GeneratorContext, pkg string, found []Sentinel, types []ErrType) sdk.Node {
	if len(found) > 0 {
		for _, v := range ctx.Reader.Variables().Slice() {
			if v.Package == pkg && v.Name == found[0].Name {
				return v
			}
		}
	}
	return types[0].decl
}

// prefixOf resolves what every sentinel's message must begin with, or empty
// when the check is suppressed.
func prefixOf(pkg *sdk.Package) string {
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
func neighboursOf(ctx *sdk.GeneratorContext, pkg *sdk.Package, byPackage map[string][]Sentinel) []Neighbour {
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
		// The package clause, not the path's last element: `example.com/other/v2`
		// is package `other`, because the major-version suffix belongs to the
		// module path and not to the name. [golang.PackageName] applies that rule
		// and path.Base does not, so a neighbour imported at v2 would render in
		// the generated file's list as `v2`.
		out = append(out, Neighbour{Path: p, Name: golang.PackageName(p), Sentinels: byPackage[p]})
	}
	return out
}

// errTypesOf lifts every exported type in pkg that implements error.
//
// Matched on SHAPE, not on name. A type declaring `Error()` with no result is
// not an error, and a generated check calling it as one does not compile — so
// asking whether the name is present is the wrong question, and the one that
// puts a build failure in the consumer's repository rather than a diagnostic in
// ours. The optional halves are matched the same way: `Is` and `Unwrap` earn a
// check only at the signature [errors.Is] and [errors.Unwrap] actually consult.
func errTypesOf(ctx *sdk.GeneratorContext, pkg string) []ErrType {
	var out []ErrType
	for _, s := range ctx.Reader.Structs().Slice() {
		if s.Package != pkg || !golang.IsExported(s.Name) {
			continue
		}
		methods, unresolved := methodSetOf(ctx, s)
		if !golang.ImplementsError(methods) {
			continue
		}
		fields, fieldEmbeds := golang.ExportedFieldSet(s, ctx.Reader)
		reportUnresolved(ctx, s, unresolved, fieldEmbeds)

		out = append(out, ErrType{
			Name:       s.Name,
			Ref:        sdk.NewExternal(s.Package, s.Name),
			Pointer:    sdk.PointerReceiver(methods, ErrorMethod),
			HasIs:      golang.IsIsMethod(sdk.MethodByName(methods, IsMethod)),
			HasUnwrap:  golang.IsUnwrapMethod(sdk.MethodByName(methods, UnwrapMethod)),
			CauseField: causeFieldOf(fields),
			Fields:     fieldsOf(fields),
			decl:       s,
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

// methodSetOf returns the struct's full method set — declared plus promoted —
// and the embeds the run could not resolve.
//
// The declared set alone is not a method set. `type NotFoundError struct {
// *BaseError; Key string }` is the dominant Go idiom for a family of custom
// errors, and reading only its declarations finds no Error method at all: the
// package's directive says its errors are a contract and the generated file
// covers half of them, silently.
//
// Promotion order is eidos's, which is Go's: a declared member shadows a
// promoted one, so nothing that generates today generates differently.
func methodSetOf(ctx *sdk.GeneratorContext, s *sdk.Struct) ([]*sdk.Method, []golang.UnresolvedEmbed) {
	promoted, unresolved := golang.PromotedMethods(s, ctx.Reader)

	out := make([]*sdk.Method, 0, len(s.Methods)+len(promoted))
	out = append(out, s.Methods...)
	for _, p := range promoted {
		out = append(out, p.Method)
	}
	return out, unresolved
}

// reportUnresolved raises an embed the run could not follow.
//
// The set is smaller than the truth when this fires, so generating against it
// quietly asserts a contract the type may not have — or omits one it does. A
// diagnostic naming the embed is the honest answer, because the cause is
// usually that the run did not load the package the embed came from, which the
// author can fix and the generator cannot.
func reportUnresolved(ctx *sdk.GeneratorContext, s *sdk.Struct, sets ...[]golang.UnresolvedEmbed) {
	// Deduped across the sets, because the method walk and the field walk stop
	// at the same embed: one unfollowed line is one thing wrong with the
	// source, and reporting it once per question asked reads as two.
	seen := make(map[string]struct{})
	for _, embeds := range sets {
		for _, e := range embeds {
			if _, dup := seen[e.Written]; dup {
				continue
			}
			seen[e.Written] = struct{}{}
			ctx.Diag.Warnf(s.Pos(),
				"%s: %q embeds %q, which this run did not resolve, so its error contract is "+
					"checked against a method set smaller than the type's",
				Name, s.Name, e.Written)
		}
	}
}

// causeFieldOf names the exported field holding the wrapped error, or empty
// when the type declares none.
//
// The first such field rather than the only one: a type carrying two has no
// answer to "which did you mean", and picking the first is what [sdk.FieldOfType]
// documents. The name is what a composite literal needs, so the field is
// flattened to it here rather than carried into the projection.
func causeFieldOf(fields []golang.PromotedField) string {
	for _, f := range fields {
		if !literalSettable(f) {
			continue
		}
		if golang.IsError(f.Field.Type) {
			return f.Field.Name
		}
	}
	return ""
}

// literalSettable reports whether a composite literal in the generated file can
// set the field directly.
//
// Only a declared field can. A promoted one is named by a selector —
// `BaseError.Cause` — and a selector is not a composite-literal key: setting it
// needs the literal nested one level per embed, and the template does not spell
// that. Emitting the selector anyway produces `invalid field name
// BaseError.Cause in struct literal` in the consumer's build.
//
// So the method set is walked through embeds and the field set is not. The
// asymmetry is deliberate: promotion is what makes the contract visible at all,
// and the literal is a separate problem with a separate answer.
func literalSettable(f golang.PromotedField) bool {
	return f.Depth == 0 && !f.ThroughPointer && f.Field.Type != nil
}

// fieldsOf lifts the exported fields whose values a message is expected to
// carry.
func fieldsOf(fields []golang.PromotedField) []Field {
	out := make([]Field, 0, len(fields))
	for _, f := range fields {
		// See [literalSettable] for why a promoted field is skipped.
		if !literalSettable(f) {
			continue
		}
		sample := ""
		if golang.IsString(f.Field.Type) {
			sample, _ = golang.SampleValues(f.Field.Type.Name, f.Field.Name)
		}
		out = append(out, Field{Name: f.Field.Name, Sample: sample})
	}
	return out
}
