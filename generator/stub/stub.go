// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package stub

import (
	"fmt"
	"io/fs"
	"slices"
	"strings"
	"text/template"

	"go.thesmos.sh/eidos/emit"
	"go.thesmos.sh/eidos/lang/golang"
	"go.thesmos.sh/eidos/node"
	"go.thesmos.sh/eidos/plugins/annotator/shape"
	"go.thesmos.sh/eidos/sdk"

	"go.thesmos.sh/testkit/generator/internal/signature"
)

// Name is the plugin's stable identifier.
const Name = "stub"

// Capability is the label the plugin advertises so a downstream
// consumer can declare a documentary dependency on stub generation.
const Capability = "stub"

// DirectiveName is the bare directive name — without the `//testkit:`
// prefix — the plugin reads from source interfaces.
const DirectiveName sdk.DirectiveName = "stub"

// SlotName is the [emit.File] slot both emit values append into.
// `top` renders between the package clause and the first core decl,
// which is where a template-rendered block of whole declarations
// belongs.
const SlotName = "top"

// KindStub and KindStubTests are the plugin-defined emit kinds. The
// backend resolves a template by the kind's string value, so each
// constant doubles as the name the matching template defines.
const (
	KindStub      sdk.Kind = "stub.double"
	KindStubTests sdk.Kind = "stub.test"
)

// Version composes into the pipeline's plugin fingerprint, which frontends
// fold into their cache keys — so a change here invalidates a warm cache
// populated when this plugin behaved differently. A plugin declaring no
// version contributes an empty string and can never invalidate anything,
// which is a silent staleness bug waiting for its first behavioural change.
//
// Bump it on any change to what this plugin emits — the Go projection or the
// templates alike.
//
// It is deliberately a constant rather than a digest of the templates, even
// though a digest would invalidate automatically. The version renders into
// every generated file's `Plugins:` header, so a content-derived one would
// churn the header of every output in every consuming repository on any
// template edit, and a golden diff would stop isolating what actually changed
// in the output. Stability in the header is worth the discipline; during
// development `--no-cache` covers the gap.
const Version = "1.0.0"

// DefaultSuffix is the trailer appended to the source interface's
// name to form the stub type's identifier.
const DefaultSuffix = "Stub"

// langGo is the backend language identifier the per-language
// adapters key on. Every dispatcher below compares against it, so a
// second language arrives as one more arm rather than a scattered
// string literal.
const langGo = "golang"

// Options carries the plugin's user-tunable settings.
//
// Recording is not optional: it is what the suite, bench, and model tiers
// read to assert on interactions rather than only on return values. A
// toggle would let a consumer silently disable every tier above.
type Options struct {
	// Suffix overrides the stub type's name suffix. Empty falls back
	// to [DefaultSuffix].
	Suffix string `eidos:"suffix,default=Stub"`
}

// Plugin is the stub generator. The zero value is unusable; go
// through [New] so the embedded holder binds to the options field.
type Plugin struct {
	*sdk.Holder[Options]
	opts Options
}

// New returns a fresh plugin instance with the options holder bound.
func New() *Plugin {
	p := &Plugin{}
	p.Holder = sdk.BindOptions(&p.opts)
	return p
}

// Name returns [Name].
func (*Plugin) Name() string { return Name }

// Version returns [Version].
func (*Plugin) Version() string { return Version }

// Priority places the plugin in the foundation bucket: the stub is a
// base type other generators may decorate, so it must exist before
// composition and cross-cutting plugins run.
func (*Plugin) Priority() sdk.Priority { return sdk.GeneratorFoundation }

// Provides advertises [Capability].
func (*Plugin) Provides() []string { return []string{Capability} }

// Requires returns nil — the plugin reads source interfaces and
// depends on no other plugin's contribution.
func (*Plugin) Requires() []string { return nil }

// Directives declares the `//testkit:stub` schema.
//
// The directive takes no positional argument and denies negation: a
// stub exists exactly where one is declared, so deleting the line is
// the suppression and a negated form would have nothing to act on.
func (*Plugin) Directives() []sdk.DirectiveSchema {
	return []sdk.DirectiveSchema{
		sdk.NewDirective(DirectiveName).
			Describe(
				"Generates a recording test double for the annotated interface, " +
					"plus a companion test file proving the double satisfies it. " +
					"Takes no arguments. The negated form is rejected — a stub " +
					"exists only where declared, so removing the directive is the " +
					"suppression.",
			).
			On(node.KindInterface).
			DenyNegation().
			Build(),
	}
}

// Outputs dispatches to the per-language adapter. Adding a language
// adds an arm here; unknown languages return nil, which the
// framework reads as "no routable output for this backend".
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

// TemplateOverrides returns nil — the plugin replaces no canonical
// funcmap entry.
func (*Plugin) TemplateOverrides(string) template.FuncMap { return nil }

// suffix returns the configured stub-type suffix, or the documented
// default when unset.
func (p *Plugin) suffix() string {
	if p.opts.Suffix != "" {
		return p.opts.Suffix
	}
	return DefaultSuffix
}

// Method is one rendered interface method.
//
// The signature itself — parameters, returns, locals, and the named-return
// decision — comes from [signature], because the suite, bench, and model
// generators project the same source the same way. What this type adds is the
// naming convention specific to a double.
type Method struct {
	Name string

	// CallType is the identifier of the per-method recorded-call
	// struct — `<Iface><Method>Call`.
	CallType string

	// StubType is the identifier of the per-method configuration point —
	// `<Iface><Method>Stub`. It embeds the runtime's MethodStub, which is
	// what supplies recording, fault injection, latency, and gates.
	StubType string

	// ReturnType is the identifier of the fixed-return holder —
	// `<Iface><Method>Return`.
	//
	// Exported because the generated companion lives in the external test
	// package and names it when boxing a call's answer, and because a
	// consumer writing their own assertions against a recorded answer needs
	// the same. Returns remains how it is configured.
	ReturnType string

	// OnField is the field on the aggregate double that exposes this
	// method's configuration — `On<Method>`.
	OnField string

	Params  []signature.Param
	Returns []signature.Return

	// NamedReturns reports whether the generated signature declares
	// its return names. See [signature.NamedReturnsUsable] for why this is
	// all-or-nothing rather than per-return.
	NamedReturns bool

	// Shape is the detector classification the annotator stamped, empty when
	// the signature matched no detector. It names what the method *is* —
	// `reader`, `writer`, `deleter` — which is what decides the extra
	// configuration a double can usefully offer for it.
	Shape string

	// Iterator classifies the method's return as a range-over-func sequence,
	// empty when it returns none. A sequence-returning method gains Yields
	// helpers, because building one by hand in every test is a closure a
	// caller should not have to write.
	Iterator signature.Iterator

	// IteratorElem is the sequence's element type, nil when the method
	// returns no sequence.
	IteratorElem emit.Ref

	// IteratorSecond is a Seq2's second type argument, nil for a Seq or for a
	// method returning no sequence.
	IteratorSecond emit.Ref

	// IteratorYieldsError reports the `iter.Seq2[V, error]` shape, where a
	// helper can usefully append a terminal failure after the values.
	IteratorYieldsError bool

	// OrderAfter is the prerequisite method this one may only follow, taken
	// from the orderafter mixin's `fn` parameter. Empty when unconstrained.
	OrderAfter string

	// Mixins are the opt-in behavioural laws the source attached through
	// `//testkit:mixin <name>`, in the order they were written.
	//
	// Read from the annotator rather than declared as directives of this
	// plugin's own: the mixin vocabulary belongs to the shape annotator, the
	// corpus gate measures coverage against it, and a second declaration here
	// would be free to drift from it.
	Mixins []string
}

// HasMixin reports whether the source attached the named mixin, which is how
// the template decides whether to emit the configuration that mixin implies.
func (m Method) HasMixin(name string) bool { return slices.Contains(m.Mixins, name) }

// Mixin names this plugin reads. The shape annotator owns the vocabulary;
// these are the subset that changes what a double emits, as opposed to the
// ones that only state a law for the suite and model tiers.
const (
	// MixinIntegrationOnly marks a method a double cannot stand in for — it
	// reaches a real dependency, so a recorded stand-in would be a lie.
	MixinIntegrationOnly = "integrationonly"

	// MixinDeprecated marks a method whose use should be reported.
	MixinDeprecated = "deprecated"

	// MixinOrderAfter marks a method that may only be called once its
	// prerequisite has been.
	MixinOrderAfter = "orderafter"
)

// MixinOrderAfterParam is the key carrying the prerequisite method's name, as
// in `//testkit:mixin orderafter fn=Prepare`.
const MixinOrderAfterParam = "fn"

// HasResults reports whether the method returns anything, which decides
// whether a fixed-return holder and its Returns setter are emitted at all.
func (m Method) HasResults() bool { return len(m.Returns) > 0 }

// ErrReturn returns the method's error slot, or nil when it has none.
//
// Fault injection stamps the injected error onto that slot before recording
// and returns it, so the dispatch body needs to name the field rather than
// assume the error is last.
func (m Method) ErrReturn() *signature.Return {
	for i, r := range m.Returns {
		if r.Error {
			return &m.Returns[i]
		}
	}
	return nil
}

// Stub is the emit value rendered into the primary output.
type Stub struct {
	sdk.BaseEmit

	// TypeName is the stub struct's identifier — `<Iface><Suffix>`.
	TypeName string

	// IfaceName is the source interface's identifier, used in the
	// generated doc comments.
	IfaceName string

	// IfaceRef qualifies the source interface for DelegateTo's parameter.
	//
	// A double routed into its own package through `out=` no longer shares a
	// package with the interface it doubles, so the reference has to carry
	// one. Where the two do share a package the backend renders it bare.
	IfaceRef *emit.Expr

	Methods []Method

	// TypeParams is the source interface's type-parameter list, in the form
	// `renderTypeParams` spells as `[K comparable, V any]`. Empty for a
	// non-generic interface, where the helper renders nothing.
	TypeParams []*emit.TypeParam

	// TypeArgs is the same list in use position — `[K, V]`, or empty. Every
	// generated identifier that names one of the double's own types has to
	// carry it, since a generic type cannot be referenced bare.
	TypeArgs string

	// Complete reports whether the double carries every method its source
	// declares. An integration-only method is dropped, and a double missing a
	// method cannot satisfy its interface — so the compile-time assertion is
	// emitted only when nothing was dropped.
	Complete bool
}

// Generic reports whether the double is parameterised, which is what decides
// whether a companion can be generated for it at all — a Go test function
// cannot take type parameters, so the checks have no way to name the types the
// double is instantiated at.
func (s *Stub) Generic() bool { return len(s.TypeParams) > 0 }

// Ordered reports whether any method carries an order constraint, which is
// what decides whether the double allocates a tracker at all.
func (s *Stub) Ordered() bool {
	// Indexed rather than ranged by value: Method is wide enough that copying
	// one per iteration is measurable, and nothing here needs a copy.
	for i := range s.Methods {
		if s.Methods[i].OrderAfter != "" {
			return true
		}
	}
	return false
}

// Kind returns [KindStub].
func (*Stub) Kind() sdk.Kind { return KindStub }

// Tests is the emit value rendered into the tagged test output.
//
// The companion always lands in an external test package — the
// framework appends `_test` to whatever package the primary output
// resolved to — so it can never reach either type it exercises
// unqualified. Both are carried as [sdk.NewExternal] expressions and
// the backend registers the qualifying imports.
//
// The two references resolve from different places, and the
// difference is the whole reason [Tests] implements
// [emit.OutputPackageSetter]:
//
//   - IfaceRef names the source interface, which is hand-written and
//     stays where the author put it. Its package is known during
//     Generate.
//   - StubRef names the stub this plugin generates, which follows
//     `out=` / `pkg=` routing. Its package is not decided until
//     Layout, so it is filled in by [Tests.SetOutputPackages].
type Tests struct {
	sdk.BaseEmit

	TypeName  string
	IfaceName string

	// IfaceRef qualifies the source interface. Set during Generate.
	IfaceRef *emit.Expr

	// CtorRef qualifies the double's constructor, which lives beside it and
	// therefore follows the same routing.
	CtorRef *emit.Expr

	// StubRef qualifies the generated stub. Set during Generate
	// against the source package as a provisional value, then
	// corrected by [Tests.SetOutputPackages] once routing resolves.
	// The provisional value is what a run without a Layout phase —
	// a direct generator unit test — observes.
	StubRef *emit.Expr

	Methods []Method

	// Generic mirrors [Stub.Generic]. A parameterised double gets a note in
	// place of its checks rather than no file at all: the absence has to be
	// visible to a reader who expected one, and an empty file would read as a
	// generator that failed silently.
	Generic bool

	// Complete mirrors [Stub.Complete], so the companion's compile-time
	// assertion is emitted on exactly the same condition as the double's.
	Complete bool
}

// Complete mirrors [Stub.Complete], so the companion's compile-time
// assertion is emitted on exactly the same condition as the double's.

// Kind returns [KindStubTests].
func (*Tests) Kind() sdk.Kind { return KindStubTests }

// SetOutputPackages repoints [Tests.StubRef] at wherever Layout
// routed the primary output.
//
// The companion is always the external test package of the primary
// (`<pkg>_test`), so the reference is always qualified — there is no
// routing under which the stub and its test share a package, and
// therefore no case where the correct rendering is a bare name.
//
// An empty path for the primary tag means the Target resolved
// without a derivable import path, which centralised routing does.
// The provisional source-package reference is left in place rather
// than replaced with an unqualified name: a wrong package is a
// compile error naming the symbol, while a bare name silently binds
// to whatever else is in scope.
func (t *Tests) SetOutputPackages(byTag map[string]string) {
	if path := byTag[""]; path != "" {
		t.StubRef = sdk.NewExternal(path, t.TypeName)
		t.CtorRef = sdk.NewExternal(path, "New"+t.TypeName)
	}
}

// Generate walks every source interface carrying `//testkit:stub` and
// queues one [Stub] against the primary output and one [Tests]
// against the tagged test output. The Layout phase resolves each
// contribution's target; both land beside the source interface by
// default and follow directive / config / CLI overrides otherwise.
//
// Interfaces without the directive are skipped silently. An
// annotated interface with no methods is skipped with a positioned
// diagnostic — a double with no behaviour to stand in for is
// certainly a mistake, and emitting an empty struct would hide it.
func (p *Plugin) Generate(ctx *sdk.GeneratorContext) error {
	c := sdk.NewProvenance(Name, sdk.EmitTarget{})
	for _, iface := range ctx.Reader.Interfaces().Slice() {
		if !iface.HasPositiveDirective(DirectiveName) {
			continue
		}
		typeName := iface.Name + p.suffix()
		methods := methodsOf(iface)

		// Measured after projection rather than on the source method set: an
		// interface whose every method is integration-only has methods but
		// nothing a double can stand in for, and emitting the shell would
		// produce a type that satisfies nothing and records nothing.
		if len(methods) == 0 {
			ctx.Diag.Errorf(iface.Pos(),
				"%s: interface %q carries //testkit:%s but has no methods a double can stand in for",
				Name, iface.QName(), DirectiveName)
			continue
		}

		base := sdk.BaseEmit{
			OriginNode: iface,
			SetByName:  c.SetBy(),
			SourcePos:  iface.Pos(),
		}
		testBase := base
		testBase.OutputTagName = GoTestOutputTag

		// Appended through one loop rather than two blocks. The pair differs
		// only in its emit kind and output tag, and a second copy of the
		// append-and-check dance is where the two would drift apart.
		for _, value := range []sdk.EmitNode{
			&Stub{
				BaseEmit:   base,
				TypeName:   typeName,
				IfaceName:  iface.Name,
				IfaceRef:   sdk.NewExternal(iface.Package, iface.Name),
				Methods:    methods,
				TypeParams: typeParamsOf(iface),
				TypeArgs:   typeArgsOf(iface),
				Complete:   complete(iface, methods),
			},
			&Tests{
				BaseEmit:  testBase,
				TypeName:  typeName,
				IfaceName: iface.Name,
				StubRef:   sdk.NewExternal(iface.Package, typeName),
				CtorRef:   sdk.NewExternal(iface.Package, "New"+typeName),
				IfaceRef:  sdk.NewExternal(iface.Package, iface.Name),
				Methods:   methods,
				Generic:   len(iface.TypeParams) > 0,
				Complete:  complete(iface, methods),
			},
		} {
			prov := c.Provenance(string(value.Kind()) + "." + iface.Name)
			if err := ctx.Store.Emit().AppendOriginSlot(iface, SlotName, value, prov); err != nil {
				return fmt.Errorf("%s: append %s slot for %q: %w", Name, value.Kind(), iface.Name, err)
			}
		}
	}
	return nil
}

// complete reports whether the double carries every method its source
// declares, which is what decides whether the compile-time interface assertion
// is emitted.
//
// An embedded interface fails it outright. The projection reads only the
// directly declared methods, so a double for an interface that embeds another
// is missing whatever the embedded one contributed — and asserting that it
// satisfies the interface would fail in the consumer's compiler rather than
// here. Flattening embedded method sets is the fix; until then the double is
// honestly partial rather than confidently wrong.
func complete(iface *node.Interface, methods []Method) bool {
	return len(iface.Embeds) == 0 && len(methods) == len(iface.Methods)
}

// typeParamsOf lifts the interface's type-parameter list into the emit form
// `renderTypeParams` consumes.
//
// Written here rather than taken from lang/golang because that package's
// helper is typed to a struct, and an interface's list is the same shape for a
// different host.
func typeParamsOf(iface *node.Interface) []*emit.TypeParam {
	if len(iface.TypeParams) == 0 {
		return nil
	}
	out := make([]*emit.TypeParam, len(iface.TypeParams))
	for i, p := range iface.TypeParams {
		out[i] = &emit.TypeParam{
			Name:       p.Name,
			Constraint: golang.ConstraintFromNode(p.Constraint),
		}
	}
	return out
}

// typeArgsOf returns the use form of the interface's type-parameter list —
// `[K, V]` — or empty for a non-generic interface, so a template can append it
// unconditionally.
func typeArgsOf(iface *node.Interface) string {
	if len(iface.TypeParams) == 0 {
		return ""
	}
	names := make([]string, len(iface.TypeParams))
	for i, p := range iface.TypeParams {
		names[i] = p.Name
	}
	return "[" + strings.Join(names, ", ") + "]"
}

// iteratorReturn returns the method's sole return when it has exactly one,
// which is the only shape a sequence helper can drive.
//
// A method returning `(iter.Seq[V], error)` is deliberately not treated as a
// sequence: the helper would have to invent the error's value on every call,
// and inventing return values is what makes a double lie.
func iteratorReturn(m *node.Method) *node.TypeRef {
	if len(m.Returns) != 1 {
		return nil
	}
	return m.Returns[0].Type
}

// orderAfter reads the prerequisite method from the orderafter mixin's
// parameter, or returns empty when the method carries no such constraint.
func orderAfter(m *node.Method) string {
	if !slices.Contains(shape.Mixins(m.Meta()), MixinOrderAfter) {
		return ""
	}
	name, _ := shape.MixinParamKey(MixinOrderAfter, MixinOrderAfterParam).Get(m.Meta())
	return name
}

// doubled reports whether a double can stand in for m.
//
// An integration-only method reaches a real dependency — a network, a
// database — so a recorded stand-in would answer for something it never did.
// Dropping it means the double no longer satisfies its interface, which is
// deliberate and why the compile-time assertion is guarded on completeness.
func doubled(m *node.Method) bool {
	return !slices.Contains(shape.Mixins(m.Meta()), MixinIntegrationOnly)
}

// methodsOf lifts every method a double carries into the rendered form both
// outputs share. Free function rather than a method: the lifting depends only
// on the source signature and the annotator's stamps, not on plugin options.
func methodsOf(iface *node.Interface) []Method {
	out := make([]Method, 0, len(iface.Methods))
	for _, m := range iface.Methods {
		if !doubled(m) {
			continue
		}
		params := signature.ParamsOf(m)
		named := signature.NamedReturnsUsable(m)
		out = append(out, Method{
			Name:                m.Name,
			CallType:            iface.Name + m.Name + "Call",
			StubType:            iface.Name + m.Name + "Stub",
			ReturnType:          iface.Name + m.Name + "Return",
			OnField:             "On" + m.Name,
			Params:              params,
			Returns:             signature.WithLocals(signature.ReturnsOf(m), params, named),
			NamedReturns:        named,
			Shape:               shape.Get(m.Meta()),
			Mixins:              shape.Mixins(m.Meta()),
			OrderAfter:          orderAfter(m),
			Iterator:            signature.IteratorOf(iteratorReturn(m)),
			IteratorElem:        signature.IteratorElem(iteratorReturn(m)),
			IteratorSecond:      signature.IteratorSecond(iteratorReturn(m)),
			IteratorYieldsError: signature.IteratorYieldsError(iteratorReturn(m)),
		})
	}
	return out
}
