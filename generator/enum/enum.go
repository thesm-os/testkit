// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package enum

import (
	"fmt"
	"io/fs"
	"sort"
	"strconv"
	"strings"
	"text/template"

	"go.thesmos.sh/eidos/emit"
	"go.thesmos.sh/eidos/lang/golang"
	"go.thesmos.sh/eidos/node"
	"go.thesmos.sh/eidos/pipeline"
	"go.thesmos.sh/eidos/sdk"
)

// Name is the plugin's stable identifier.
const Name = "enum"

// Capability is the label the plugin advertises so a downstream consumer can
// declare a documentary dependency on enum generation.
const Capability = "enum"

// DirectiveName opts a typed constant block in.
const DirectiveName sdk.DirectiveName = "enum"

// ValueName pins one variant's textual form:
//
//	PillAspirin Pill = iota //testkit:value Aspirin
//
// For a variant whose derived spelling clashes with a protocol's, which is the
// case the derivation cannot be taught about.
//
// Read but not declared: `value` is one of the framework's own directives
// ([pipeline.ValueDirective]), registered for every run. A plugin declaring it
// again fails the build, since a directive name may be registered once.

// MethodsKey suppresses method generation entirely, leaving the checks:
//
//	//testkit:enum methods=off
//
// For a type whose surface is already written by hand and only wants pinning.
// A single method that exists is skipped without this — the key is for
// declaring the intent up front rather than discovering it method by method.
const MethodsKey = "methods"

// MethodsOff is the only value [MethodsKey] accepts.
const MethodsOff = "off"

// The methods the plugin generates, each skipped when the type already
// declares it. The names are the API, so they are constants rather than
// literals scattered through the projection and the template.
const (
	MethodString      = "String"
	MethodMarshal     = "MarshalText"
	MethodUnmarshal   = "UnmarshalText"
	MethodIsValid     = "IsValid"
	ParsePrefix       = "Parse"
	ValuesSuffix      = "Values"
	SentinelPrefix    = "ErrUnknown"
	unknownTextSample = "__testkit_unknown__"
)

// SlotName is the [emit.File] slot both outputs land in.
const SlotName = "top"

// KindAPI and KindTests are the plugin-defined emit kinds. The backend resolves
// a template by the kind's string value, so each constant doubles as the name
// its template defines.
const (
	KindAPI   sdk.Kind = "enum.api"
	KindTests sdk.Kind = "enum.test"
)

// Version composes into the pipeline's plugin fingerprint.
const Version = "1.0.0"

// langGo is the backend language identifier the per-language adapters key on.
const langGo = golang.Language

// Form is how a variant renders as text.
type Form string

const (
	// FormIdentifier renders the variant's own name, which for a numeric enum
	// is the only textual form the declaration carries.
	FormIdentifier Form = "identifier"

	// FormValue renders the declared value. A string enum's value *is* its
	// textual form, and it is already written down — deriving a different one
	// discards the only thing the declaration said, and breaks every value
	// that arrives from JSON, a database column or a query parameter.
	FormValue Form = "value"
)

// Plugin is the enum generator.
type Plugin struct{}

// New returns a fresh plugin instance.
func New() *Plugin { return &Plugin{} }

// Name returns [Name].
func (*Plugin) Name() string { return Name }

// Version returns [Version].
func (*Plugin) Version() string { return Version }

// Priority places the plugin in the foundation bucket: an enum's API is a base
// a later generator may read, so it exists before composition runs.
func (*Plugin) Priority() sdk.Priority { return sdk.GeneratorFoundation }

// Provides advertises [Capability].
func (*Plugin) Provides() []string { return []string{Capability} }

// Requires returns nil.
func (*Plugin) Requires() []string { return nil }

// Directives declares both schemas.
func (*Plugin) Directives() []sdk.DirectiveSchema {
	return []sdk.DirectiveSchema{
		sdk.NewDirective(DirectiveName).
			Describe(
				"Generates an enum's textual and validity surface — String, " +
					"Parse, MarshalText, UnmarshalText, Values, IsValid — plus " +
					"checks over the declared set. A method the type already " +
					"declares is never generated; `methods=off` suppresses all of " +
					"them and leaves the checks. The negated form is rejected: " +
					"removing the directive is the suppression.",
			).
			AllowedKeys(MethodsKey).
			On(node.KindEnum).
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

// TemplateOverrides returns nil.
func (*Plugin) TemplateOverrides(string) template.FuncMap { return nil }

// Variant is one declared constant.
type Variant struct {
	// Name is the identifier.
	Name string

	// Ref qualifies it, so the checks can name it from the external test
	// package the framework routes them into.
	Ref *emit.Expr

	// Text is the variant's textual form, as a Go string literal ready to
	// render — quoted here rather than in the template so the quoting rule
	// lives beside the derivation that decides it.
	Text string
}

// API is the emit value rendered into the primary output.
type API struct {
	sdk.BaseEmit

	TypeName string
	TypeRef  *emit.Expr

	// Underlying names the enum's base type, which decides whether the
	// fallback converts numerically or textually.
	Underlying string

	// PackageName is the enum's own package, which prefixes the parse
	// sentinel's message. The package rather than the type: a message is read
	// in a log beside messages from everywhere else, and what a reader needs
	// first is which package raised it. The type name goes in the body, where
	// it distinguishes this sentinel from its neighbours.
	PackageName string

	Form     Form
	Variants []Variant

	// ParseName, ValuesName and SentinelName are the derived identifiers, held
	// here rather than composed in the template so a rename is one edit.
	ParseName, ValuesName, SentinelName string

	// Generate lists the methods this run emits — every one the type does not
	// already declare, or none when the directive said so.
	Generate map[string]bool
}

// Emits reports whether the named method is this run's to write.
func (a *API) Emits(method string) bool { return a.Generate[method] }

// Any reports whether anything at all is generated, which decides whether the
// primary file is worth emitting.
func (a *API) Any() bool { return len(a.Generate) > 0 }

// Kind returns [KindAPI].
func (*API) Kind() sdk.Kind { return KindAPI }

// Tests is the emit value rendered into the tagged test output.
type Tests struct {
	sdk.BaseEmit

	TypeName string
	TypeRef  *emit.Expr
	Form     Form
	Variants []Variant

	// ParseRef, ValuesRef and SentinelRef qualify the API this file drives.
	// The checks are routed into the external test package, so nothing in the
	// source package is reachable unqualified.
	ParseRef, ValuesRef, SentinelRef *emit.Expr

	// ZeroName is the variant whose value is the zero, empty when none is. The
	// two cases read as opposite assertions, and which one an enum earns is
	// the thing a fixture pair exists to tell apart.
	ZeroName string

	// OutOfRange is a value past the declared set as source text, empty when
	// none could be derived. Used to check that an undeclared value does not
	// render as a declared one.
	OutOfRange string

	// Each reports whether the surface a check drives actually exists.
	//
	// Parses and Marshals track what this run generated rather than what the
	// type has: a package-level Parse the author wrote is invisible to the
	// enum node, so a check assuming it would name a function that may not be
	// there. Renders and Validates are methods, so a hand-written one is
	// visible and counts.
	Renders, Parses, Marshals, Encodes, Validates, Enumerates bool
}

// Kind returns [KindTests].
func (*Tests) Kind() sdk.Kind { return KindTests }

// Generate walks every enum carrying the directive and queues its API and
// checks.
func (*Plugin) Generate(ctx *sdk.GeneratorContext) error {
	c := sdk.NewProvenance(Name, sdk.EmitTarget{})
	for _, e := range ctx.Reader.Enums().Slice() {
		if !e.HasPositiveDirective(DirectiveName) {
			continue
		}
		if len(e.Variants) == 0 {
			ctx.Diag.Errorf(e.Pos(),
				"%s: enum %q carries //testkit:%s but declares no variant",
				Name, e.QName(), DirectiveName)
			continue
		}
		if stray := strayVariants(ctx, e); len(stray) > 0 {
			ctx.Diag.Errorf(e.Pos(),
				"%s: enum %q has constants of its type declared in %s; "+
					"move them beside the type or the generated set will exclude them",
				Name, e.QName(), strings.Join(stray, ", "))
			continue
		}
		form := formOf(e)
		variants := variantsOf(e, form)
		if dup := duplicateText(variants); dup != "" {
			ctx.Diag.Errorf(e.Pos(),
				"%s: enum %q renders two variants as %s; pin one with //testkit:%s",
				Name, e.QName(), dup, pipeline.ValueDirective)
			continue
		}
		api := &API{
			BaseEmit:     sdk.BaseEmit{OriginNode: e, SetByName: c.SetBy(), SourcePos: e.Pos()},
			TypeName:     e.Name,
			TypeRef:      sdk.NewExternal(e.Package, e.Name),
			Underlying:   underlyingOf(e),
			PackageName:  packageName(ctx, e.Package),
			Form:         form,
			Variants:     variants,
			ParseName:    ParsePrefix + e.Name,
			ValuesName:   e.Name + ValuesSuffix,
			SentinelName: SentinelPrefix + e.Name,
			Generate:     generated(e),
		}
		tests := &Tests{
			BaseEmit:    baseFor(api, GoTestOutputTag),
			TypeName:    api.TypeName,
			TypeRef:     api.TypeRef,
			Form:        form,
			Variants:    variants,
			ParseRef:    sdk.NewExternal(e.Package, api.ParseName),
			ValuesRef:   sdk.NewExternal(e.Package, api.ValuesName),
			SentinelRef: sdk.NewExternal(e.Package, api.SentinelName),
			ZeroName:    zeroOf(e, variants),
			OutOfRange:  outOfRange(e, form, variants),
			Renders:     api.Emits(MethodString) || declares(e, MethodString),
			Parses:      api.Emits(ParsePrefix),
			Marshals:    api.Emits(MethodMarshal) && api.Emits(MethodUnmarshal),
			Encodes:     api.Emits(MethodMarshal) || declares(e, MethodMarshal),
			Validates:   api.Emits(MethodIsValid) || declares(e, MethodIsValid),
			Enumerates:  api.Emits(ValuesSuffix),
		}
		for _, emitted := range queued(api, tests) {
			prov := c.Provenance(string(emitted.Kind()) + "." + e.Name)
			if err := ctx.Store.Emit().AppendOriginSlot(e, SlotName, emitted, prov); err != nil {
				return fmt.Errorf("%s: append %s slot for %q: %w", Name, emitted.Kind(), e.Name, err)
			}
		}
	}
	return nil
}

// queued returns the emit values this enum contributes.
//
// The API is skipped when nothing is left to generate — a type declaring every
// method already, or one that asked for none. An empty file carrying only a
// generated-by header reads as a generator that failed.
func queued(api *API, tests *Tests) []sdk.EmitNode {
	if !api.Any() {
		return []sdk.EmitNode{tests}
	}
	return []sdk.EmitNode{api, tests}
}

// baseFor derives the tagged output's emit base from the primary's.
func baseFor(api *API, tag string) sdk.BaseEmit {
	base := api.BaseEmit
	base.OutputTagName = tag
	return base
}

// strayVariants returns the packages declaring constants of e's type outside
// e's own package, sorted.
//
// Legal Go, and silently wrong here. The frontend coalesces a constant block
// into the enum it belongs to only within one package, so a constant declared
// elsewhere is invisible to the set — and every generated answer about that
// set would then be confidently false. IsValid would reject a value someone
// declared, String would fall through to the numeric form for it, and the
// exhaustiveness check would pin a count that is not the truth.
//
// Reported rather than absorbed. Reaching across a package boundary to load
// them would make the generated set depend on which packages a run happened to
// include, so the same type would generate differently from one invocation to
// the next.
func strayVariants(ctx *sdk.GeneratorContext, e *node.Enum) []string {
	seen := map[string]bool{}
	for _, c := range ctx.Reader.Constants().Slice() {
		if c.Package == e.Package || c.Type == nil {
			continue
		}
		if c.Type.Package == e.Package && c.Type.Name == e.Name {
			seen[c.Package] = true
		}
	}
	out := make([]string, 0, len(seen))
	for pkg := range seen {
		out = append(out, pkg)
	}
	sort.Strings(out)
	return out
}

// packageName returns the short name of the package at path, falling back to
// the path's last segment when the run did not load a node for it.
//
// The declared name rather than the directory: the two usually agree and
// occasionally do not, and a message naming a package that does not exist is
// worse than one naming a directory.
func packageName(ctx *sdk.GeneratorContext, path string) string {
	for _, p := range ctx.Reader.Packages().Slice() {
		if p.Path == path {
			return p.Name
		}
	}
	if i := strings.LastIndex(path, "/"); i >= 0 {
		return path[i+1:]
	}
	return path
}

// formOf decides how a variant renders as text.
func formOf(e *node.Enum) Form {
	if underlyingOf(e) == "string" {
		return FormValue
	}
	return FormIdentifier
}

// underlyingOf returns the enum's base type name, or empty when unrecorded.
func underlyingOf(e *node.Enum) string {
	if e.Underlying == nil {
		return ""
	}
	return e.Underlying.Name
}

// variantsOf lifts every declared constant with its textual form resolved.
func variantsOf(e *node.Enum, form Form) []Variant {
	out := make([]Variant, 0, len(e.Variants))
	for _, v := range e.Variants {
		out = append(out, Variant{
			Name: v.Name,
			Ref:  sdk.NewExternal(e.Package, v.Name),
			Text: strconv.Quote(textOf(e, v, form)),
		})
	}
	return out
}

// textOf resolves one variant's textual form, low to high: the derived
// spelling, then a `//testkit:value` override.
func textOf(e *node.Enum, v *node.EnumVariant, form Form) string {
	for _, dir := range v.Directives() {
		if dir.Name == pipeline.ValueDirective && len(dir.Args) > 0 {
			return dir.Args[0]
		}
	}
	if form == FormValue {
		// A string variant's value arrives in verbatim source form, quotes
		// included. Unquoting here rather than in the template keeps the one
		// place that knows the value is source text.
		if unquoted, err := strconv.Unquote(v.Value); err == nil {
			return unquoted
		}
		return v.Value
	}
	// `StatusActive` on `type Status int` renders as `Active`: the type name
	// is already context wherever the value appears, and repeating it is noise
	// in every log line and wire payload.
	return strings.TrimPrefix(v.Name, e.Name)
}

// duplicateText returns the first textual form two variants share, or empty.
//
// Reported rather than generated around: Parse maps text to exactly one
// variant, so a collision makes one of them unreachable through it, and the
// generated round-trip check would fail with no indication of the cause.
func duplicateText(variants []Variant) string {
	seen := make(map[string]bool, len(variants))
	for _, v := range variants {
		if seen[v.Text] {
			return v.Text
		}
		seen[v.Text] = true
	}
	return ""
}

// generated returns the methods this run writes: every one the type does not
// already declare, unless the directive suppressed all of them.
//
// Skipping silently rather than reporting a clash. An author who wrote their
// own String meant to keep it, and a generator that refused to run until they
// deleted it would be demanding they give up the more specific statement.
func generated(e *node.Enum) map[string]bool {
	if suppressed(e) {
		return map[string]bool{}
	}
	out := map[string]bool{}
	for _, m := range []string{MethodString, MethodMarshal, MethodIsValid} {
		if !declares(e, m) {
			out[m] = true
		}
	}
	// Parse and Values are package-level rather than methods, so a same-named
	// declaration is not something the enum node can see. They ride with
	// String: a type keeping its own String almost always keeps its own Parse,
	// and generating one that shadows theirs is the worse guess.
	if out[MethodString] {
		out[ParsePrefix] = true
		out[ValuesSuffix] = true
	}
	// UnmarshalText is written in terms of Parse, so it rides with it rather
	// than with the other methods. Generated without it, the file names a
	// function nothing declares.
	if out[ParsePrefix] && !declares(e, MethodUnmarshal) {
		out[MethodUnmarshal] = true
	}
	return out
}

// suppressed reports whether the directive asked for no methods at all.
func suppressed(e *node.Enum) bool {
	for _, dir := range e.Directives() {
		if string(dir.Name) == string(DirectiveName) && dir.KV[MethodsKey] == MethodsOff {
			return true
		}
	}
	return false
}

// declares reports whether the enum's type already has the named method.
func declares(e *node.Enum, method string) bool {
	for _, m := range e.Methods {
		if m.Name == method {
			return true
		}
	}
	return false
}

// zeroOf returns the variant whose value is the zero, or empty when none is.
func zeroOf(e *node.Enum, variants []Variant) string {
	for i, v := range e.Variants {
		if isZero(v.Value) {
			return variants[i].Name
		}
	}
	return ""
}

// isZero reports whether a variant's verbatim value is its type's zero.
func isZero(value string) bool { return value == "0" || value == `""` }

// outOfRange returns a value past the declared set as source text, or empty
// when none can be derived.
//
// Integers take one past the largest, which is the boundary a fallback is most
// likely to get wrong. Strings take a fixed marker no sensible declaration
// would collide with. Anything else — a float, whose value arrives as an exact
// rational this cannot safely render, or an unrecorded underlying type —
// yields nothing, and the check is dropped rather than written against a value
// that might be declared.
func outOfRange(e *node.Enum, form Form, variants []Variant) string {
	if form == FormValue {
		for _, v := range variants {
			if v.Text == strconv.Quote(unknownTextSample) {
				return ""
			}
		}
		return strconv.Quote(unknownTextSample)
	}
	highest, ok := largest(e)
	if !ok {
		return ""
	}
	return strconv.FormatInt(highest+1, 10)
}

// largest returns the greatest declared value, or false when any of them is
// not an integer this can read.
func largest(e *node.Enum) (int64, bool) {
	var highest int64
	for i, v := range e.Variants {
		n, err := strconv.ParseInt(v.Value, 10, 64)
		if err != nil {
			return 0, false
		}
		if i == 0 || n > highest {
			highest = n
		}
	}
	return highest, len(e.Variants) > 0
}
