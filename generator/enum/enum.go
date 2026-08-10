// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package enum

import (
	"fmt"
	"strings"

	"go.thesmos.sh/eidos/lang/golang"
	"go.thesmos.sh/eidos/sdk"
	sdkgolang "go.thesmos.sh/eidos/sdk/golang"
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
// ([golang.EnumValueDirective]), registered for every run. A plugin declaring
// it again fails the build, since a directive name may be registered once.

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

// The surface the plugin generates, each entry skipped when the type already
// declares it. The names are the API, so they are constants rather than
// literals scattered through the projection and the template.
//
// String, MarshalText and UnmarshalText are not here: [golang.MethodString],
// [golang.MethodMarshalText] and [golang.MethodUnmarshalText] hold them, and
// the name a generator emits and the name a detector looks for have to be one
// string rather than two that agree today.
const (
	// MethodIsValid has no counterpart upstream. [golang.MethodValidate] is
	// `Validate` — a different method with a different signature — so naming
	// this one through it would generate the wrong identifier.
	MethodIsValid = "IsValid"

	// ParsePrefix is the prefix [golang.ParseFuncName] composes the parse
	// function's identifier from, repeated here as the key the projection and
	// the template track that function under. The function is package-level, so
	// it is not a method name the enum node could carry.
	ParsePrefix = "Parse"

	// ValuesSuffix composes the accessor's identifier — `<Type>Values` — and
	// keys it. Go states no convention for this one, so there is nothing
	// upstream to defer to.
	ValuesSuffix = "Values"

	// SentinelSubject is the subject [golang.SentinelName] turns into
	// `ErrUnknown<Type>`, which is the spelling every sentinel detector matches
	// on.
	SentinelSubject = "Unknown"
)

// SlotName is the [sdk.EmitFile] slot both outputs land in.
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

// Plugin is the enum generator.
//
// The embedded base answers the six declaration methods — name, version,
// priority, capabilities, directives, and the Go output/template set — so the
// plugin itself carries only what makes it this generator.
type Plugin struct{ *sdkgolang.Base }

// New returns a fresh plugin instance.
//
// The templates register no helper of their own. Everything they call is
// either a backend builtin — `renderExpr`, `external` — or one of
// [golang.AllFuncMap]'s entries, which the base merges under this plugin's own
// prefix. The testkit import path a check resolves its assertions through is
// carried on the emit value instead; see [RuntimePaths].
//
// # Failure mode
//
// [sdkgolang.Builder.Build] panics on a declaration the pipeline cannot serve
// rather than returning an error. It runs here, so the failure fires in the
// first test that constructs the plugin rather than in the first run that
// renders nothing and explains why in no output at all.
func New() *Plugin {
	return &Plugin{Base: sdkgolang.NewGenerator(Name, goTemplatesFS, GoOutputs()...).
		Version(Version).
		// An enum's API is a base a later generator may read, so it exists
		// before composition runs.
		Priority(sdk.GeneratorFoundation).
		Provides(Capability).
		Directives(directives()...).
		Build()}
}

// directives declares the schema for [DirectiveName].
func directives() []sdk.DirectiveSchema {
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
			On(sdk.NodeKindEnum).
			DenyNegation().
			Build(),
	}
}

// Variant is one declared constant.
type Variant struct {
	// Name is the identifier.
	Name string

	// Ref qualifies it, so the checks can name it from the external test
	// package the framework routes them into.
	Ref *sdk.Expr

	// Text is the variant's textual form, as a Go string literal ready to
	// render — quoted by [golang.EnumTextLiteral] rather than in the template
	// so the quoting rule lives beside the derivation that decides it.
	Text string
}

// API is the emit value rendered into the primary output.
type API struct {
	sdk.BaseEmit

	TypeName string
	TypeRef  *sdk.Expr

	// Underlying is the type String's numeric fallback converts a value to
	// before printing it, and Verb is the format verb that conversion prints
	// under.
	//
	// Derived together by [golang.EnumFallback], because choosing one without
	// the other is how `fmt.Sprintf("Ratio(%d)", float64(v))` gets written —
	// output that prints `%!d(float64=0.5)` and that `go vet` reports as a
	// defect in a repository whose authors did not write it.
	//
	// A reference rather than a name, and rendered through `renderType` rather
	// than `renderExpr`: a conversion's operand is a type, and that is the path
	// registering the import. Spelled as a bare name, an enum declared over
	// another package's type emitted `Priority(v)` with nothing importing it.
	Underlying sdk.Ref
	Verb       string

	// PackageName is the enum's own package, which prefixes the parse
	// sentinel's message. The package rather than the type: a message is read
	// in a log beside messages from everywhere else, and what a reader needs
	// first is which package raised it. The type name goes in the body, where
	// it distinguishes this sentinel from its neighbours.
	PackageName string

	Form     golang.EnumForm
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
	RuntimePaths

	TypeName string
	TypeRef  *sdk.Expr
	Form     golang.EnumForm
	Variants []Variant

	// ParseRef, ValuesRef and SentinelRef qualify the API this file drives.
	// The checks are routed into the external test package, so nothing in the
	// source package is reachable unqualified.
	ParseRef, ValuesRef, SentinelRef *sdk.Expr

	// ZeroName is the variant whose value is the zero, empty when none is, and
	// ZeroRef is that same variant as a reference. The two cases read as
	// opposite assertions, and which one an enum earns is the thing a fixture
	// pair exists to tell apart.
	//
	// The reference is carried rather than the check rebuilding the variant by
	// position. Declaration order and zero-ness are different questions and
	// agree only for a set declaring its zero first — so a set written
	// `US Region = "us-east"; Unset Region = ""` asserted the zero equalled US
	// and failed in the consumer's repository, naming a variant the assertion
	// did not mention.
	ZeroName string
	ZeroRef  *sdk.Expr

	// UnknownText is the quoted text a parse-refusal probe submits, empty when
	// the declared set already contains the marker.
	UnknownText string

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
	c := sdk.NewProvenance(Name)
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
		if dup, clash := golang.DuplicateText(e); clash {
			ctx.Diag.Errorf(e.Pos(),
				"%s: enum %q renders two variants as %q; pin one with //testkit:%s",
				Name, e.QName(), dup, golang.EnumValueDirective)
			continue
		}
		variants := variantsOf(e)
		conversion, verb := golang.EnumFallback(e)
		zeroName, zeroRef := zeroVariant(e)
		api := &API{
			BaseEmit:     sdk.EmitBase(c, e),
			TypeName:     e.Name,
			TypeRef:      sdk.NewExternal(e.Package, e.Name),
			Underlying:   conversion,
			Verb:         verb,
			PackageName:  packageName(ctx, e.Package),
			Form:         golang.EnumFormOf(e),
			Variants:     variants,
			ParseName:    golang.ParseFuncName(e.Name),
			ValuesName:   e.Name + ValuesSuffix,
			SentinelName: golang.SentinelName(SentinelSubject + e.Name),
			Generate:     generated(e),
		}
		tests := &Tests{
			BaseEmit:     sdk.EmitBaseTagged(api.BaseEmit, GoTestOutputTag),
			RuntimePaths: GoRuntime(),
			TypeName:     api.TypeName,
			TypeRef:      api.TypeRef,
			Form:         api.Form,
			Variants:     variants,
			ParseRef:     sdk.NewExternal(e.Package, api.ParseName),
			ValuesRef:    sdk.NewExternal(e.Package, api.ValuesName),
			SentinelRef:  sdk.NewExternal(e.Package, api.SentinelName),
			ZeroName:     zeroName,
			ZeroRef:      zeroRef,
			UnknownText:  unknownText(e),
			OutOfRange:   outOfRange(e),
			Renders:      api.Emits(golang.MethodString) || golang.EnumDeclares(e, golang.MethodString),
			Parses:       api.Emits(ParsePrefix),
			Marshals: api.Emits(golang.MethodMarshalText) &&
				api.Emits(golang.MethodUnmarshalText),
			Encodes: api.Emits(golang.MethodMarshalText) ||
				golang.EnumDeclares(e, golang.MethodMarshalText),
			Validates:  api.Emits(MethodIsValid) || golang.EnumDeclares(e, MethodIsValid),
			Enumerates: api.Emits(ValuesSuffix),
		}
		if err := sdk.QueueEmit(ctx.Store.Emit(), c, SlotName, e, queued(api, tests)...); err != nil {
			// Wrapped even though the queue names the plugin and the slot: what
			// it cannot name is which declaration the run was on when it failed,
			// and that is the only part a reader needs to find the source line.
			return fmt.Errorf("%s: queue enum %q: %w", Name, e.Name, err)
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
func strayVariants(ctx *sdk.GeneratorContext, e *sdk.Enum) []string {
	return golang.ForeignVariants(e, ctx.Reader.Constants().Slice())
}

// packageName returns the short name of the package at path, falling back to
// the name the path implies when the run did not load a node for it.
//
// The declared name rather than the directory: the two usually agree and
// occasionally do not, and a message naming a package that does not exist is
// worse than one naming a directory. [golang.PackageName] is the fallback
// because the last segment is not always the name — `example.com/cfg/v2` is
// package `cfg`, and taking the trailing segment would report `v2`.
//
// The fallback is unreachable from a well-formed run — every enum the walk
// reaches belongs to a package the same run loaded — and is kept because the
// alternative is an empty package clause in a generated file, which fails at
// format rather than here.
func packageName(ctx *sdk.GeneratorContext, path string) string {
	p, loaded := ctx.Reader.Packages().
		Where(func(p *sdk.Package) bool { return p.Path == path }).
		First()
	if loaded {
		return p.Name
	}
	return golang.PackageName(path)
}

// variantsOf lifts every declared constant with its textual form resolved.
//
// The text arrives from [golang.EnumTextLiteral] already quoted, which is
// where the quoting belongs: an authored `value` override is arbitrary text,
// and a template concatenating quotes around one carrying a quote produces a
// literal that truncates at the first one.
func variantsOf(e *sdk.Enum) []Variant {
	out := make([]Variant, 0, len(e.Variants))
	for _, v := range e.Variants {
		out = append(out, Variant{
			Name: v.Name,
			Ref:  sdk.NewExternal(e.Package, v.Name),
			Text: golang.EnumTextLiteral(e, v),
		})
	}
	return out
}

// generated returns the methods this run writes: every one the type does not
// already declare, unless the directive suppressed all of them.
//
// Skipping silently rather than reporting a clash. An author who wrote their
// own String meant to keep it, and a generator that refused to run until they
// deleted it would be demanding they give up the more specific statement.
//
// [golang.EnumMethods] is not the source of the candidate list: it answers over
// the six shapes eidos knows, which include the JSON pair this plugin
// deliberately does not emit and exclude IsValid, which it does.
func generated(e *sdk.Enum) map[string]bool {
	if suppressed(e) {
		return map[string]bool{}
	}
	out := map[string]bool{}
	for _, m := range []string{golang.MethodString, golang.MethodMarshalText, MethodIsValid} {
		if !golang.EnumDeclares(e, m) {
			out[m] = true
		}
	}
	// Parse and Values are package-level rather than methods, so a same-named
	// declaration is not something the enum node can see. They ride with
	// String: a type keeping its own String almost always keeps its own Parse,
	// and generating one that shadows theirs is the worse guess.
	if out[golang.MethodString] {
		out[ParsePrefix] = true
		out[ValuesSuffix] = true
	}
	// UnmarshalText is written in terms of Parse, so it rides with it rather
	// than with the other methods. Generated without it, the file names a
	// function nothing declares.
	if out[ParsePrefix] && !golang.EnumDeclares(e, golang.MethodUnmarshalText) {
		out[golang.MethodUnmarshalText] = true
	}
	return out
}

// suppressed reports whether the directive asked for no methods at all.
func suppressed(e *sdk.Enum) bool {
	dir := e.Directive(DirectiveName)
	return dir != nil && dir.Value(MethodsKey) == MethodsOff
}

// zeroVariant returns the variant whose value is the zero, as a name for the
// assertion's message and a reference for the assertion itself.
//
// Both from one lookup. The check previously took the name here and rebuilt the
// variant by position in the template, which are different questions: they agree
// only for a set that declares its zero first.
func zeroVariant(e *sdk.Enum) (name string, ref *sdk.Expr) {
	v, ok := golang.ZeroVariant(e)
	if !ok {
		return "", nil
	}
	return v.Name, sdk.NewExternal(e.Package, v.Name)
}

// unknownText returns the quoted text a parse-refusal probe submits, or empty
// when the declared set already contains the marker.
//
// The same marker and the same collision check the out-of-range probe uses.
// Composed in a template before, where the collision could not be checked at
// all — a variant pinned to it produced a check that failed because the parse
// succeeded.
func unknownText(e *sdk.Enum) string {
	text, ok := golang.OutOfRangeText(e)
	if !ok {
		return ""
	}
	return golang.Quote(text)
}

// outOfRange returns a value past the declared set as source text, or empty
// when none can be derived.
//
// A string enum's probe is a marker no sensible declaration collides with, and
// the collision is checked rather than assumed. A numeric one takes a value
// outside the set whatever kind it is declared in — [golang.OutOfRangeLiteral]
// asks the integer question first, because an integral literal parses as a
// float too and a set declared over `int` means the narrower reading.
//
// Empty drops the subtest, which is the conservative answer: a probe rendered
// from a value the set turned out to declare would assert that a declared
// variant is undeclared.
func outOfRange(e *sdk.Enum) string {
	if golang.EnumFormOf(e) == golang.FormValue {
		text, ok := golang.OutOfRangeText(e)
		if !ok {
			return ""
		}
		return golang.Quote(text)
	}
	literal, ok := golang.OutOfRangeLiteral(e)
	if !ok {
		return ""
	}
	return literal
}
