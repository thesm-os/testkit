// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package suite

import (
	"fmt"
	"slices"

	"go.thesmos.sh/eidos/lang/golang"
	"go.thesmos.sh/eidos/plugins/annotator/shape"
	"go.thesmos.sh/eidos/plugins/annotator/shape/detectors/lookup"
	"go.thesmos.sh/eidos/plugins/annotator/shape/detectors/pointerreader"
	"go.thesmos.sh/eidos/plugins/annotator/shape/detectors/readernoerror"
	"go.thesmos.sh/eidos/plugins/annotator/shape/detectors/readerwithbool"
	"go.thesmos.sh/eidos/plugins/annotator/shape/mixins/deprecated"
	"go.thesmos.sh/eidos/plugins/annotator/shape/mixins/integrationonly"
	"go.thesmos.sh/eidos/plugins/annotator/shape/mixins/nilsafe"
	"go.thesmos.sh/eidos/plugins/annotator/shape/mixins/orderafter"
	"go.thesmos.sh/eidos/plugins/annotator/shape/mixins/sideeffect"
	"go.thesmos.sh/eidos/plugins/annotator/shape/mixins/timeout"
	"go.thesmos.sh/eidos/sdk"
	sdkgolang "go.thesmos.sh/eidos/sdk/golang"

	"go.thesmos.sh/testkit/generator/stub"
)

// Name is the plugin's stable identifier.
const Name = "suite"

// Capability is the label the plugin advertises, so the generators that read
// this one's projection — bench, fuzz, model — can declare the dependency.
const Capability = "suite"

// Version composes into the pipeline's plugin fingerprint. Bump it on any
// change to what this plugin emits, the projection or the templates alike.
const Version = "1.0.0"

// DirectiveName is the bare directive name — without the `//testkit:` prefix —
// that opts an interface in.
const DirectiveName sdk.DirectiveName = "suite"

// SlotName is the [sdk.EmitFile] slot the harness lands in.
const SlotName = "top"

// KindContract is the emit kind for the harness as a whole. The backend
// resolves a template by the kind's string value, so the constant doubles as
// the template's name.
const KindContract sdk.Kind = "suite.contract"

// The emit kinds for the signature-derived checks.
//
// One kind per check, because the backend's `render` builtin dispatches on
// Kind() and a template reference cannot be composed from an expression. That
// is what lets each check live in its own template file, which is the only way
// seventy-two of them stay readable.
const (
	KindSmoke       sdk.Kind = "suite.check.smoke"
	KindCancel      sdk.Kind = "suite.check.cancel"
	KindDeadline    sdk.Kind = "suite.check.deadline"
	KindNilContext  sdk.Kind = "suite.check.nilcontext"
	KindZeroOnError sdk.Kind = "suite.check.zeroonerror"
)

// The emit kinds for the detector-derived checks.
//
// One family, split by how a shape signals absence. [KindZeroOnError] is the
// third member and belongs to the signature rather than to a classification,
// because an error return says on its own that a call can fail; a bool or a bare
// zero says nothing without knowing the method is a lookup, which is what the
// shape stamp supplies (docs/adr/0018).
const (
	KindMissZero sdk.Kind = "suite.check.misszero"
	KindMissFlag sdk.Kind = "suite.check.missflag"
)

// The emit kinds for the mixin-derived checks.
const (
	KindNilSafe    sdk.Kind = "suite.check.nilsafe"
	KindTimeout    sdk.Kind = "suite.check.timeout"
	KindOrderAfter sdk.Kind = "suite.check.orderafter"
	KindSideEffect sdk.Kind = "suite.check.sideeffect"
)

// Plugin is the conformance-suite generator.
//
// The embedded [sdkgolang.Base] answers every declaration the pipeline asks for
// — name, version, priority, capabilities, directives, outputs, templates and
// the template funcmap — so the only method this package writes is
// [Plugin.Generate].
type Plugin struct{ *sdkgolang.Base }

// New returns a fresh plugin instance.
//
// # Placement
//
// [sdk.GeneratorComposition], one bucket after the double's
// [sdk.GeneratorFoundation]. The bucket is what orders the two; a Requires
// naming a plugin in an earlier bucket is silently ignored by eidos's sorter,
// so nothing here relies on one.
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
		Priority(sdk.GeneratorComposition).
		Provides(Capability).
		Directives(directives()...).
		Build()}
}

// directives declares the `//testkit:suite` schema.
//
// No positional argument: the interface the directive sits on is the subject,
// and a name beside it would be a second way to say the same thing. No keys
// either — benchmarks and fuzz targets are their own generators' directives,
// scoped to a method, because a team wants them on the paths that matter rather
// than on all six. Negation is denied because a suite exists exactly where one
// is declared, so deleting the line is the suppression (docs/adr/0016).
func directives() []sdk.DirectiveSchema {
	return []sdk.DirectiveSchema{
		sdk.NewDirective(DirectiveName).
			Describe(
				"Generates a conformance harness for the annotated interface: " +
					"an Assert<Iface>Contract entry point running every check " +
					"the signature and the interface's classifications imply, " +
					"a typed extension point per method, and the inputs those " +
					"checks need, derived. Takes no argument — the interface it " +
					"sits on is the subject. Benchmarks and fuzz targets are " +
					"//testkit:bench and //testkit:fuzz, which scope to a method.",
			).
			On(sdk.NodeKindInterface).
			DenyNegation().
			Build(),
	}
}

// Subject is what every emit value in this file needs to name the interface it
// is about.
//
// Embedded in both the harness and each check rather than reached through a
// back-pointer from one to the other: the emit graph is walked and versioned,
// and a cycle in it is a hazard for the sake of three strings.
type Subject struct {
	// IfaceName is the source interface's identifier, which names every
	// generated declaration.
	IfaceName string

	// IfaceRef qualifies the interface. The harness is routed into its own
	// package in the ordinary case, where it is not reachable unqualified.
	//
	// A reference rather than an expression: this appears in type position —
	// a parameter, a struct field — and the two render through different
	// builtins. [sdk.NewExternal] builds the expression form, which is what a
	// call site needs and what a type position rejects. [sdk.External] is the
	// reference form and would serve; [golang.RefFor] is taken instead because
	// it answers for a predeclared name too, so one call covers every type an
	// interface could be named by.
	IfaceRef sdk.Ref

	// Runtime is testkit's module root, where the assertion helpers the
	// generated checks call live. The backend's `external` builtin turns a path
	// and a symbol into a qualified reference and registers the import, so a
	// path is all a template needs.
	Runtime string

	// ClockRef is [clock.Clock] in type position — a config field, a parameter.
	//
	// Built here rather than composed in a template, because `external` yields
	// the expression form and a type position rejects it. The two render
	// through different builtins, and the mismatch is a render error rather
	// than a compile one, so it surfaces as a file that came out short.
	ClockRef sdk.Ref

	// TypeParams is the source interface's type-parameter list in declaration
	// form, which `renderTypeParams` spells as `[K comparable, V any]`. Empty
	// for a non-generic interface, where the helper renders nothing.
	TypeParams []*sdk.EmitTypeParam

	// TypeArgs is the same list in use position — `[K, V]`, or empty.
	//
	// Every generated identifier naming a type that carries parameters has to
	// carry it too, since a generic type cannot be referenced bare. That
	// includes the subject: `Store` alone is not a type, `Store[K, V]` is.
	TypeArgs string
}

// Check is one generated assertion.
//
// One type rather than one per kind, with the kind carried as a field: the data
// a check needs is uniform — which method, what it reports as, which exported
// assertion it calls, which derived input it is handed — and what differs
// between seventy-two of them is the template, which is exactly what keying the
// dispatch on [Check.Kind] selects.
type Check struct {
	sdk.BaseEmit
	Subject

	// KindName is the emit kind, and therefore the name of the template that
	// renders this check. Composed once, in Go, from the classification the
	// projection read.
	KindName sdk.Kind

	// Subtest is what the check reports under, nested inside its method's
	// group. Descriptive for a structural property, the classification's own
	// name for a classification (docs/adr/0015).
	Subtest string

	// Func is the exported assertion's identifier — `AssertMixedStoreCancels`.
	// Exported so a consumer can run one standalone, and so the companion file
	// can drive it against a stand-in and prove it fails.
	Func string

	// Path is the drop key a consumer names in `<Iface>Without` —
	// `Store/reports a cancelled context`. Composed from the method and the
	// subtest so it reads as what the failure output shows.
	Path string

	// NeedsDerivedInput reports whether the check's meaning depends on the
	// values it is handed rather than merely on their existing.
	//
	// Most do not. Cancellation, an expired deadline and a nil context are
	// claims about the context, and a smoke call asks only that the method
	// survive one — every one of them holds with the zero value, which every
	// type has. Only a check whose semantics are "this input must miss" needs a
	// value derivation could reach.
	//
	// The distinction is what keeps a method taking a callback, a channel or a
	// slice from losing its whole family: eidos derives no literal for those,
	// and dropping every check that mentions one costs four checks that never
	// looked at it.
	NeedsDerivedInput bool

	// NeedsClock reports whether the check reads time, and so is handed the
	// run's [clock.Clock] rather than reaching for the wall clock.
	//
	// Generated code that calls time.Now is generated code whose subject is
	// partly the machine it runs on. A consumer builds their implementation in
	// the factory they supply, so one that takes a clock can be built with the
	// same [clock.TestClock] the harness measures on — and then "within a
	// budget" is a claim about the time the implementation means to spend,
	// settled deterministically, rather than about how loaded the machine was.
	//
	// The default is the real clock, so an implementation that does not take
	// one behaves as it would have.
	NeedsClock bool

	// Args names the fixture fields this check is handed, one per parameter the
	// method takes after its context.
	//
	// Field names rather than values, because the fixture is one struct a
	// consumer may replace whole: a check holding a literal would keep running
	// against the derived value after an override replaced it.
	Args []string

	// Method is the signature under check.
	Method Method

	// Partner is the second callable a relational classification names, with
	// PartnerArgs the identifiers a call to it is handed.
	//
	// A signature rather than a name, because the check calls it: `sideeffect`
	// observes before and after, `hooks` registers through it. The name alone
	// was enough for `orderafter`, which asserts that calling early fails and
	// never calls the partner at all.
	Partner     *Method
	PartnerArgs []string
}

// Kind returns [Check.KindName].
func (c *Check) Kind() sdk.Kind { return c.KindName }

// Method is one method of the subject interface, with the naming this generator
// adds to the shared signature projection.
type Method struct {
	// Sig is the source signature in rendered form, embedded so `.Name`,
	// `.Params`, `.Returns` and `.ReturnsError` promote onto the method.
	//
	// From [golang.Sig] rather than derived here, because every Go generator
	// projects the same source the same way and four independent
	// implementations had already disagreed about it.
	*golang.Sig

	// CheckType is the identifier of this method's extension point —
	// `<Iface><Method>Check`. Every generated check for the method is a value
	// of it, and so is a consumer's, which is what lets them compose.
	CheckType string

	// ArgFields names the fixture field each of the method's non-context
	// parameters is supplied from, in order.
	//
	// The fixture's names rather than the parameters' own, because two methods
	// naming one parameter at different types get a field each. Carried on the
	// method so the extension point's call site and the generated checks read
	// one answer: they did not, and a consumer's check was handed the other
	// method's value.
	ArgFields []string

	// Checks are the generated assertions for this method, in the order the
	// entry point runs them.
	Checks []*Check

	// Mixins names the classifications the annotator attached, and Contracts
	// the same for contract roles.
	//
	// Carried on the projection rather than read from the source node at each
	// use. A check is selected once and rendered later, and the node is not in
	// scope by then — but more to the point, two derivations of the same stamp
	// are two chances to disagree about what the run classified.
	Mixins, Contracts []string

	// mixinParams holds each attached mixin's KV arguments, keyed
	// `<mixin>.<param>`.
	//
	// Unexported with an accessor, because a template reaching a map by a
	// composed key would spell the composition itself — and the one thing
	// worth hiding here is that a sibling param arrives qualified and has to be
	// cut back down to the local name a generated call can use.
	mixinParams map[string]string
}

// HasMixin reports whether the annotator attached the named classification.
func (m Method) HasMixin(name string) bool { return slices.Contains(m.Mixins, name) }

// MixinParam returns a mixin's KV argument, and whether one was written.
//
// The value verbatim. A param the mixin declares as a sibling arrives as a
// qualified name, which is right for identity and wrong for a call site — see
// [Method.MixinPartner] for that.
func (m Method) MixinParam(name, param string) (string, bool) {
	v, ok := m.mixinParams[name+"."+param]
	return v, ok
}

// MixinPartner returns the local identifier a mixin's sibling param names.
//
// The shape resolver rewrites a sibling param into a qualified name so it is
// unambiguous across packages. Generated code calls the partner on the subject
// it already holds, so the qualified form is exactly what it cannot spell —
// [golang.LocalName] takes the trailing identifier back off, which is what
// [generator/stub] already does for `orderafter`.
func (m Method) MixinPartner(name, param string) string {
	v, _ := m.MixinParam(name, param)
	return golang.LocalName(v)
}

// TakesContext reports whether the method's first parameter is a context.
//
// The gate on three of the five signature-derived checks: cancellation, an
// expired deadline and a nil context are all claims about a parameter a method
// may not take, and emitting them for one that does not would not compile.
func (m Method) TakesContext() bool {
	return len(m.Params) > 0 && golang.IsContext(m.Params[0].Source)
}

// CallArgs returns the parameters a generated call passes after the context,
// which is every parameter for a method that takes none.
func (m Method) CallArgs() []golang.Param {
	if m.TakesContext() {
		return m.Params[1:]
	}
	return m.Params
}

// HasInput reports whether the method takes anything after its context.
//
// The only lever a harness has over a subject. A parameterless method can still
// fail — a closed store, a dropped connection — but not because of anything the
// suite chose, so a check whose meaning is "this input misses" cannot reach the
// failure it is about and would demand one from a correct implementation.
func (m Method) HasInput() bool { return len(m.CallArgs()) > 0 }

// VariadicParam returns the method's variadic parameter, or nil.
//
// Go allows at most one and only in final position, so one answer covers the
// signature. Present so the generated file can state a narrowing a reader would
// otherwise have to infer: the fixture derives one value per parameter, so a
// generated check calls a variadic method with exactly one element.
func (m Method) VariadicParam() *golang.Param {
	for i := range m.Params {
		if m.Params[i].Variadic {
			return &m.Params[i]
		}
	}
	return nil
}

// ValueReturns returns the result slots that are not the error, which is what
// a zero-value check compares.
func (m Method) ValueReturns() []golang.Return {
	out := make([]golang.Return, 0, len(m.Returns))
	for _, r := range m.Returns {
		if !r.Error {
			out = append(out, r)
		}
	}
	return out
}

// FlagReturn returns the trailing bool a comma-ok shape signals absence with,
// or nil.
//
// Trailing rather than anywhere, because that is what the idiom is: a bool in
// any other position is a value the method computed, not a report on whether it
// found anything. Nil for every other signature, which is what the templates
// branch on.
func (m Method) FlagReturn() *golang.Return {
	values := m.ValueReturns()
	if len(values) == 0 {
		return nil
	}
	last := &values[len(values)-1]
	if golang.QName(last.Source) != "bool" {
		return nil
	}
	return last
}

// MissReturns returns the slots a miss check holds to their zero: every value
// return except the flag that reported the miss.
//
// The flag is excluded because it is the signal rather than a result — asserting
// it is zero is asserting `false == false`, and a check that cannot fail is
// worse than no check.
func (m Method) MissReturns() []golang.Return {
	values := m.ValueReturns()
	if m.FlagReturn() != nil {
		return values[:len(values)-1]
	}
	return values
}

// Double names the generated stand-in a harness runs itself through.
//
// Read off the double's own queued emit value rather than composed here. The
// identifiers are the stub generator's convention, and a second derivation
// would be free to name a symbol it never emitted.
type Double struct {
	// TypeName is the double's identifier.
	TypeName string

	// CtorName constructs one, and DelegateToName is the option that makes it
	// forward to a real implementation.
	CtorName, DelegateToName string
}

// Contract is the emit value rendered into the primary output.
type Contract struct {
	sdk.BaseEmit
	Subject

	// EntryName is the identifier a consumer calls — `Assert<Iface>Contract`.
	EntryName string

	// Fixture is the derived input set every check is handed values from.
	Fixture Fixture

	// Seed is the write each fresh subject is populated through, nil for an
	// interface declaring no writer.
	Seed *Seed

	// Double is the generated stand-in for this interface, nil where the source
	// declared no `//testkit:stub`.
	//
	// Its presence makes the harness run twice: once against the subject and
	// once against the same subject wrapped in the double. Anything the wrapper
	// fails that the subject passes is the double lying — which is what makes a
	// generated double trustworthy, and it is a run a consumer should not have
	// to write out.
	Double *Double

	Methods []Method
}

// Kind returns [KindContract].
func (*Contract) Kind() sdk.Kind { return KindContract }

// CheckCount reports how many checks the harness runs, for the header a reader
// meets before them.
func (c *Contract) CheckCount() int {
	var n int
	for _, m := range c.Methods {
		n += len(m.Checks)
	}
	return n
}

// Generate queues one harness per interface carrying the directive.
func (*Plugin) Generate(ctx *sdk.GeneratorContext) error {
	c := sdk.NewProvenance(Name)
	doubles := sdk.PendingByOrigin[*stub.Stub](ctx.Store.Emit())
	for _, iface := range ctx.Reader.Interfaces().Slice() {
		if !iface.HasPositiveDirective(DirectiveName) {
			continue
		}
		set, complete := resolveMethods(ctx, iface)
		if !complete {
			continue
		}
		if len(set.Methods) == 0 {
			ctx.Diag.Errorf(iface.Pos(),
				"%s: interface %q carries //testkit:%s but declares no method, "+
					"so the harness would assert nothing",
				Name, iface.Name, DirectiveName)
			continue
		}
		// Only the emptiness of the *method set* is checked. An interface whose
		// every check was dropped for want of an input used to be the other
		// half of this, and is no longer reachable: a check is dropped only
		// when its meaning is the value it is handed, and smoke never is.
		methods := methodsOf(iface, set)
		fixture := fixtureOf(ctx, iface, methods)
		methods = withChecks(c, ctx, iface, fixture, methods)
		if err := sdk.QueueEmit(ctx.Store.Emit(), c, SlotName, iface, &Contract{
			BaseEmit:  sdk.EmitBase(c, iface),
			Subject:   subjectOf(iface),
			EntryName: "Assert" + iface.Name + "Contract",
			Fixture:   fixture,
			Seed:      seedOf(fixture, methods),
			Double:    doubleOf(doubles, iface),
			Methods:   methods,
		}); err != nil {
			// Wrapped even though the queue names the plugin and the slot: what
			// it cannot name is which declaration the run was on when it failed,
			// and that is the only part a reader needs to find the source line.
			return fmt.Errorf("%s: queue interface %q: %w", Name, iface.Name, err)
		}
	}
	return nil
}

// resolveMethods returns the interface's full method set and whether it is
// complete.
//
// Resolution is [sdk.StoreReader.MethodSet]: the embed walk, the duplicate
// rule, the cycle guard and the attribution of a method to the embed it arrived
// through are facts about a Go method set. What is decided here is what a
// conformance harness does with an incomplete one — refuse, because a harness
// covering part of a contract reports success for an implementation that does
// not satisfy the rest.
//
// Severity splits on whether a wider run would fix it, matching the double's
// rule for the same question: an unloaded embed is a warning, because a narrow
// invocation is legitimate; a non-interface or parameterised embed is a source
// defect no wider run repairs.
func resolveMethods(ctx *sdk.GeneratorContext, iface *sdk.Interface) (sdk.MethodSetResult, bool) {
	set := ctx.Reader.MethodSet(iface)
	complete := true
	for _, issue := range set.Issues {
		// Spelled the way the source wrote it — `io.Closer`, not the bare
		// `Closer` the reference carries — so a diagnostic names something the
		// author can search for.
		written := golang.Display(issue.Embed.Type)
		switch issue.Reason {
		case sdk.ReasonCyclic:
			// Illegal in Go and unreachable from a real frontend. The walk broke
			// the cycle only after the interface it points back at had already
			// contributed, so the set is short of nothing.
			ctx.Diag.Warnf(issue.Embed.Pos(),
				"%s: interface %q embeds %q through a cycle; the walk broke out of it, "+
					"so the harness covers whatever the source had already contributed",
				Name, iface.QName(), written)
		case sdk.ReasonUnresolved:
			complete = false
			ctx.Diag.Warnf(issue.Embed.Pos(),
				"%s: interface %q embeds %q, which this run did not load, so its method "+
					"set cannot be completed; nothing is generated, because a harness "+
					"over part of a contract passes an implementation that fails the rest",
				Name, iface.QName(), written)
		default:
			complete = false
			ctx.Diag.Errorf(issue.Embed.Pos(),
				"%s: interface %q embeds %q, which %s; nothing is generated, because a "+
					"harness over part of a contract passes an implementation that fails "+
					"the rest",
				Name, iface.QName(), written, issue.Reason)
		}
	}
	return set, complete
}

// methodsOf projects every method of the resolved set, with its checks
// selected.
//
// Driven off the resolved set rather than the declarations: an interface that
// embeds another declares none of what it inherits, and a harness reading only
// declarations would cover half a contract without saying it had.
func methodsOf(iface *sdk.Interface, set sdk.MethodSetResult) []Method {
	out := make([]Method, 0, len(set.Methods))
	for _, src := range set.Methods {
		bag := src.Meta()
		out = append(out, Method{
			Sig:         golang.SigOf(src),
			CheckType:   iface.Name + src.Name + "Check",
			Mixins:      shape.Mixins(bag),
			Contracts:   shape.Contracts(bag),
			mixinParams: mixinParamsOf(bag),
		})
	}
	return out
}

// mixinParamsOf reads the KV arguments of every mixin this generator acts on.
//
// Pulled once, into a map, rather than reached through [shape.MixinParamKey] at
// each use. The projection is what a check is selected from and what a template
// renders, and neither holds the source node by then.
//
// Enumerated rather than discovered because eidos exposes no "every param
// stamped under this mixin" accessor — [shape.MixinParamKey] composes one key
// from a pair. A classification this generator does not act on has nothing to
// read, so the list is the set of checks rather than an inventory of eidos.
func mixinParamsOf(bag *sdk.Bag) map[string]string {
	wanted := [...]struct{ mixin, param string }{
		{MixinOrderAfter, MixinOrderAfterParam},
		{MixinTimeout, MixinTimeoutParam},
		{MixinSideEffect, MixinSideEffectParam},
	}
	var out map[string]string
	for _, w := range wanted {
		v, found := shape.MixinParamKey(w.mixin, w.param).Get(bag)
		if !found {
			continue
		}
		if out == nil {
			out = make(map[string]string, len(wanted))
		}
		out[w.mixin+"."+w.param] = v
	}
	return out
}

// signatureChecks selects the family a method owes for its signature alone.
//
// None of these needs a directive, and together they are most of the volume: the
// three-method validates fixture owes ten before a single classification is
// read, and its one directive adds the eleventh.
// They are also the half no [engine/model/law] property covers, which is what
// makes them unambiguously this tier's (docs/adr/0018).
func signatureChecks(c *sdk.Provenance, iface *sdk.Interface, f Fixture, m Method) []*Check {
	// The sample values for the ordinary checks, and the second set for the one
	// that has to miss. A zero-value check called with the value the subject was
	// seeded with succeeds, and then asserts nothing.
	args := fixtureArgs(f, m, false)
	other := fixtureArgs(f, m, true)

	base := func(kind sdk.Kind, subtest, suffix string, with []string, derived bool) *Check {
		return &Check{
			BaseEmit:          sdk.EmitBase(c, iface),
			Subject:           subjectOf(iface),
			KindName:          kind,
			Subtest:           subtest,
			Func:              "Assert" + iface.Name + m.Name + suffix,
			Path:              m.Name + "/" + subtest,
			Args:              with,
			NeedsDerivedInput: derived,
			Method:            m,
		}
	}

	out := []*Check{base(KindSmoke, "smoke", "Smoke", args, false)}
	if m.TakesContext() && m.ReturnsError() {
		out = append(out,
			base(KindCancel, "reports a cancelled context", "Cancels", args, false),
			base(KindDeadline, "reports an expired deadline", "HonoursDeadline", args, false),
		)
	}
	if m.TakesContext() {
		out = append(out, base(KindNilContext, "tolerates a nil context", "ToleratesNilContext", args, false))
	}
	if m.ReturnsError() && len(m.ValueReturns()) > 0 && m.HasInput() {
		// The only one whose meaning is in the value: a miss check called with
		// the value the subject was seeded with succeeds, and asserts nothing.
		//
		// HasInput because the check reaches the failure it is about through the
		// alternate value, and a method taking nothing after its context leaves
		// nowhere to put one. Emitted anyway it fatals against every correct
		// implementation, and names a fixture field that does not exist.
		out = append(out, base(KindZeroOnError, "an error carries the zero value", "ZeroOnError", other, true))
	}
	return out
}

// The mixins this generator generates a check for, and the parameters it reads.
//
// Named here rather than taken from eidos's own constants at each use so the
// set a reader has to hold is one list. Each is suite-owned under
// docs/adr/0018 — no [engine/model/law] property covers it — and each is
// derivable from the stamp plus the signature, which is what excludes the rest:
// `validates` and `scope` need a value no run can invent, and `sideeffect`,
// `partition`, `hooks` and `sample` name a partner eidos declares no parameter
// for (thesm-os/eidos#16).
const (
	MixinNilSafe         = nilsafe.Name
	MixinDeprecated      = deprecated.Name
	MixinIntegrationOnly = integrationonly.Name
	MixinTimeout         = timeout.Name
	MixinTimeoutParam    = timeout.ParamDuration
	MixinOrderAfter      = orderafter.Name
	MixinOrderAfterParam = orderafter.ParamFn
	MixinSideEffect      = sideeffect.Name
	MixinSideEffectParam = sideeffect.ParamObserve
)

// missShapes are the classifications whose absence signal is a value rather
// than an error, and which no [engine/model/law] property covers.
//
// A stamp rather than the signature, which is the one judgement here. `(T,
// bool)` and a bare `T` are common shapes that are not always lookups — a
// `Validate(v) (Report, bool)` returns a verdict, and holding its report to the
// zero when the flag is false would be a check about nothing. The stamp is what
// says the method answers a question about presence (docs/adr/0018).
//
// The cost is that an identically shaped method eidos classifies otherwise gets
// no check. Reversing this to a signature gate is a one-line change if that
// trade turns out wrong.
var missShapes = map[string]struct{}{
	readernoerror.Name:  {},
	readerwithbool.Name: {},
	lookup.Name:         {},
	pointerreader.Name:  {},
}

// detectorChecks selects the family a method owes for its classification.
//
// Only the miss family so far, and only where the shape says absence is
// reported in a value. The error-signalled member of the same family is
// [KindZeroOnError], which the signature earns on its own.
//
// HasInput for the same reason zero-on-error needs it: the miss is reached by
// choosing an input that is not there, and a method taking nothing after its
// context offers nowhere to put one.
func detectorChecks(c *sdk.Provenance, iface *sdk.Interface, f Fixture, m Method) []*Check {
	if _, owned := missShapes[shape.Get(m.Source.Meta())]; !owned || !m.HasInput() {
		return nil
	}
	if len(m.MissReturns()) == 0 {
		// Nothing to hold to a zero. A lookup whose only result is the flag
		// reports absence and returns nothing else, so the check would assert
		// that `false` is `false`.
		return nil
	}

	kind, suffix := KindMissZero, "ReportsAMiss"
	if m.FlagReturn() != nil {
		kind = KindMissFlag
	}
	return []*Check{{
		BaseEmit:          sdk.EmitBase(c, iface),
		Subject:           subjectOf(iface),
		KindName:          kind,
		Subtest:           "reports a miss",
		Func:              "Assert" + iface.Name + m.Name + suffix,
		Path:              m.Name + "/reports a miss",
		Args:              fixtureArgs(f, m, true),
		NeedsDerivedInput: true,
		Method:            m,
	}}
}

// mixinChecks selects the family a method owes for the classifications attached
// to it.
//
// Three, where the RFC's tier table lists sixteen. What is missing is not
// unwritten: `validates` and `scope` need a value no run can invent, four name a
// partner eidos declares no parameter for (thesm-os/eidos#16), `concurrent` and
// `concurrentreaders` assert nothing without `-race`, `retrysucceeds` has no
// attempt count to read, `integrationonly` is a build tag rather than an
// assertion, and `deprecated` is a fact about a method rather than a claim about
// its behaviour — it is stated in the generated documentation instead.
// partnerArgs names the identifiers a call to the partner is handed, and
// whether the check can spell them all.
//
// A generated check receives the *annotated* method's parameters and nothing
// else, so the partner can only be called with values already in scope. Both
// resolve their parameters to fixture fields, and a field they share is one
// identifier serving both — which is the ordinary case, since a partner
// observing an effect keyed on something observes it by the same key.
//
// Where the partner needs a field the method does not take, the check is not
// generated: widening its parameter list would give it a signature no other
// check has, for a shape the corpus does not contain.
func partnerArgs(f Fixture, m, partner Method) ([]string, bool) {
	byField := map[string]string{}
	for _, p := range m.CallArgs() {
		byField[f.FieldFor(p)] = p.Name
	}
	out := make([]string, 0, len(partner.CallArgs()))
	for _, p := range partner.CallArgs() {
		name, found := byField[f.FieldFor(p)]
		if !found {
			return nil, false
		}
		out = append(out, name)
	}
	return out, true
}

// partnerOf resolves a relational mixin's sibling param to the method it names.
//
// Against the interface's own resolved method set, so an inherited partner is
// found: the resolver guarantees the name refers to something in scope, and a
// conformance check can only call what the subject declares.
//
// Nil when the mixin names none. The param is optional by design — a bare
// `//testkit:mixin sideeffect` is still a classification — so its absence is a
// check not generated rather than a fault to report.
func partnerOf(methods []Method, m Method, mixin, param string) *Method {
	name := m.MixinPartner(mixin, param)
	if name == "" {
		return nil
	}
	for i := range methods {
		if methods[i].Name == name {
			return &methods[i]
		}
	}
	return nil
}

func mixinChecks(
	c *sdk.Provenance, iface *sdk.Interface, f Fixture, m Method, methods []Method,
) []*Check {
	base := func(kind sdk.Kind, subtest, suffix string, args []string) *Check {
		return &Check{
			BaseEmit: sdk.EmitBase(c, iface),
			Subject:  subjectOf(iface),
			KindName: kind,
			Subtest:  subtest,
			Func:     "Assert" + iface.Name + m.Name + suffix,
			Path:     m.Name + "/" + subtest,
			Args:     args,
			Method:   m,
		}
	}

	var out []*Check
	if m.HasMixin(MixinNilSafe) && m.HasInput() {
		// The check supplies its own zeros, so it takes no argument — but it
		// needs a parameter to zero, and a method taking none has nothing to
		// be unsafe about.
		out = append(out, base(KindNilSafe, MixinNilSafe, "IsNilSafe", nil))
	}
	if _, declared := m.MixinParam(MixinTimeout, MixinTimeoutParam); declared {
		// Gated on the parameter rather than the mixin: "within a budget" is
		// not a statement until a duration is named, and a bare
		// `//testkit:mixin timeout` names none.
		ck := base(KindTimeout, MixinTimeout, "CompletesInBudget", fixtureArgs(f, m, false))
		ck.NeedsClock = true
		out = append(out, ck)
	}
	if p := partnerOf(methods, m, MixinSideEffect, MixinSideEffectParam); p != nil && p.ReturnsError() {
		// The observation is the check, so a partner that cannot report its own
		// failure leaves the comparison unable to tell "unchanged" from "the
		// observer broke".
		if args, spellable := partnerArgs(f, m, *p); spellable {
			ck := base(KindSideEffect, MixinSideEffect, "HasAnObservableEffect", fixtureArgs(f, m, false))
			ck.Partner, ck.PartnerArgs = p, args
			out = append(out, ck)
		}
	}
	if p := m.MixinPartner(MixinOrderAfter, MixinOrderAfterParam); p != "" && m.ReturnsError() {
		// ReturnsError because the claim is that calling early *fails*, and a
		// method with nowhere to report failure cannot make it.
		out = append(out, base(KindOrderAfter, MixinOrderAfter, "RequiresItsPrerequisite",
			fixtureArgs(f, m, false)))
	}
	return out
}

// doubleOf names the stand-in queued for this interface, or nil.
//
// Read from the emit queue rather than from the source directive: a directive
// says what was asked for, and the queue says what was produced. A double the
// stub generator declined to emit — an interface whose method set it could not
// complete — leaves nothing to run through, and a harness composing the call
// anyway would not compile.
func doubleOf(queued map[sdk.Node]*stub.Stub, iface *sdk.Interface) *Double {
	d, hosted := queued[sdk.Node(iface)]
	if !hosted {
		return nil
	}
	return &Double{
		TypeName:       d.TypeName,
		CtorName:       d.CtorName,
		DelegateToName: d.DelegateToName,
	}
}

// subjectOf names the interface every emit value for it is about.
func subjectOf(iface *sdk.Interface) Subject {
	return Subject{
		IfaceName:  iface.Name,
		IfaceRef:   golang.RefFor(iface.Name, iface.Package),
		Runtime:    Module,
		ClockRef:   golang.RefFor("Clock", Module+"/clock"),
		TypeParams: golang.TypeParamDecls(iface.TypeParams),
		TypeArgs:   golang.TypeParamNames(iface.TypeParams),
	}
}

// fixtureArgs names the fixture field per parameter the method takes after its
// context, taking the second value of each when alternate is set.
func fixtureArgs(f Fixture, m Method, alternate bool) []string {
	args := m.CallArgs()
	out := make([]string, 0, len(args))
	for _, p := range args {
		// The fixture's own name for the field, not the parameter's: two
		// methods naming one parameter at different types get one field each,
		// and a check has to reach its own.
		name := f.FieldFor(p)
		if alternate {
			name += OtherSuffix
		}
		out = append(out, name)
	}
	return out
}
