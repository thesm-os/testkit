// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package suite

import (
	"fmt"
	"slices"
	"strings"

	"go.thesmos.sh/eidos/lang/golang"
	"go.thesmos.sh/eidos/plugins/annotator/shape"
	"go.thesmos.sh/eidos/plugins/annotator/shape/detectors/batchreader"
	"go.thesmos.sh/eidos/plugins/annotator/shape/detectors/lookup"
	"go.thesmos.sh/eidos/plugins/annotator/shape/detectors/pointerreader"
	"go.thesmos.sh/eidos/plugins/annotator/shape/detectors/readernoerror"
	"go.thesmos.sh/eidos/plugins/annotator/shape/detectors/readerwithbool"
	"go.thesmos.sh/eidos/plugins/annotator/shape/mixins/concurrent"
	"go.thesmos.sh/eidos/plugins/annotator/shape/mixins/deprecated"
	"go.thesmos.sh/eidos/plugins/annotator/shape/mixins/hooks"
	"go.thesmos.sh/eidos/plugins/annotator/shape/mixins/idempotent"
	"go.thesmos.sh/eidos/plugins/annotator/shape/mixins/integrationonly"
	"go.thesmos.sh/eidos/plugins/annotator/shape/mixins/lifecycleafterclose"
	"go.thesmos.sh/eidos/plugins/annotator/shape/mixins/nilsafe"
	"go.thesmos.sh/eidos/plugins/annotator/shape/mixins/orderafter"
	"go.thesmos.sh/eidos/plugins/annotator/shape/mixins/partition"
	"go.thesmos.sh/eidos/plugins/annotator/shape/mixins/sample"
	"go.thesmos.sh/eidos/plugins/annotator/shape/mixins/sideeffect"
	"go.thesmos.sh/eidos/plugins/annotator/shape/mixins/timeout"
	"go.thesmos.sh/eidos/plugins/annotator/shape/mixins/validates"
	"go.thesmos.sh/eidos/plugins/annotator/shape/mixins/wrappedvia"
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
const Version = "1.2.0"

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

	// KindCloseIdempotent asserts a declared-idempotent teardown answers the
	// same on the second call; KindUseAfterClose that operations report the
	// declared sentinel once teardown ran; KindConcurrentSmoke that a
	// declared-concurrent method survives parallel callers under -race.
	KindCloseIdempotent sdk.Kind = "suite.check.closeidempotent"
	KindUseAfterClose   sdk.Kind = "suite.check.useafterclose"
	KindConcurrentSmoke sdk.Kind = "suite.check.concurrentsmoke"
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
	KindPartition  sdk.Kind = "suite.check.partition"
	KindHooks      sdk.Kind = "suite.check.hooks"
	KindSample     sdk.Kind = "suite.check.sample"
	KindValidates  sdk.Kind = "suite.check.validates"
	KindWrappedVia sdk.Kind = "suite.check.wrappedvia"
)

// KindBatchSize The emit kind for the remaining detector-derived check.
//
// Separate from the miss family because it is about arity rather than about
// absence: a batch read answers per key, and a subject answering once for many
// is wrong in a way no zero comparison reaches.
const KindBatchSize sdk.Kind = "suite.check.batchsize"

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

	// IntegrationEnv is the variable a run sets to include integration-only
	// checks. See [GoIntegrationEnv].
	IntegrationEnv string

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

	// Extra names fixture fields the check needs beyond the method's own
	// parameters, each with the identifier it binds to in the signature.
	//
	// A check is ordinarily handed one value per parameter, which is enough
	// when the claim is about a single call. Isolation is not: two partitions
	// that never interfere is a statement about two writes, and the second one
	// needs a value the method's own parameter list cannot carry.
	Extra []ExtraArg

	// SecondCall names the identifiers a two-write check passes to its second
	// call, in declaration order and without the context.
	//
	// Separate from Extra because the two answer different questions: Extra is
	// what the signature gains, SecondCall is what the call spells. Deriving
	// one from the other in a template meant asking which identifiers carried a
	// suffix, which is a rule reconstructed from spelling rather than read.
	SecondCall []string

	// CompareAgainst is the identifier a two-write check holds its read up to:
	// the payload written where the read should not reach.
	//
	// Named here rather than indexed out of Extra in a template, because
	// arithmetic in a template is a rule a reader has to reconstruct and the
	// backend's function set has nothing to do it with.
	CompareAgainst string

	// Callback is the func-typed parameter a registration partner takes, in
	// the form a generated literal has to spell.
	//
	// Present because a hook check has to *construct* the thing it registers,
	// not merely name it: the callback's own signature is what a func literal
	// declares, and nothing else in the projection carries it.
	Callback *CallbackSig

	// Partner is the second callable a relational classification names, with
	// PartnerArgs the identifiers a call to it is handed.
	//
	// A signature rather than a name, because the check calls it: `sideeffect`
	// observes before and after, `hooks` registers through it. The name alone
	// was enough for `orderafter`, which asserts that calling early fails and
	// never calls the partner at all.
	Partner     *Method
	PartnerArgs []string

	// Sentinel is the resolver-qualified error a directive names — what a
	// use-after-close operation must report once teardown ran.
	Sentinel *sdk.Expr
}

// Kind returns [Check.KindName].
func (c *Check) Kind() sdk.Kind { return c.KindName }

// CallbackSig is a func-typed parameter's own signature.
//
// Types without names, which is all a func literal needs: `func(string) {}` is
// legal Go, and inventing identifiers for parameters the body ignores would put
// names in generated source that answer to nothing in the source it came from.
type CallbackSig struct {
	// Name is the identifier the registration partner declares the parameter
	// under, used only in the diagnostic.
	Name string

	// Params and Returns are the callback's own signature.
	Params, Returns []sdk.Ref
}

// ExtraArg is one fixture field a check takes beyond the method's parameters.
type ExtraArg struct {
	// Name is the identifier the generated signature binds it to, and Field
	// the fixture field the entry point reads it from.
	Name, Field string

	// Type is the parameter's type, rendered through the backend so the file
	// registers whatever import it needs.
	Type sdk.Ref
}

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

	// IntegrationOnly reports that this method reaches something outside the
	// process, so its checks run only where that something exists.
	//
	// Carried on the projection rather than asked of the mixin list in a
	// template, because the template's job is to spell the guard and not to
	// know which classification implies one.
	IntegrationOnly bool

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

	// contractRoles holds the role this method fills in each contract it belongs
	// to, and contractPartners the role-keyed partners beside it.
	//
	// Two maps rather than one, because the axis keys its stamps two ways and
	// flattening them would need a discriminator this would have to invent.
	// Unexported with accessors for the reason mixinParams is: the composed key
	// is spelling, and a template should ask a question rather than build one.
	//
	// The third stamp a contract can carry — an opaque param — is read by
	// nothing here: every check calls what it names, and a param is by
	// definition a value with no callable in it.
	contractRoles, contractPartners map[string]string
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

	// Witnesses are the concrete types the double's own companion instantiates
	// at — pinned by the stub directive's witness key or derived from an open
	// constraint — and empty for a non-generic interface. The falsification
	// guards run at the same types: a Test function cannot carry type
	// parameters, so these are what make a generic harness provable.
	Witnesses []sdk.Ref
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

	// Coverage is every classification the interface carries, with the tier
	// that covers it — which the header states before the checks.
	Coverage []Coverage

	// Unfalsifiable says why no companion output was generated, empty where one
	// was.
	//
	// Stated in the harness a reader meets rather than left to the absence of a
	// file, for the reason an uncovered classification is stated: a missing
	// companion is indistinguishable from a generator that failed to write one.
	Unfalsifiable string

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

// Unchecked is every classification the interface declares that this file does
// not assert.
//
// What a consumer needs from the header is not which half of testkit covers
// what — they have no reason to know testkit has halves — but whether the file
// forgot something and what to do about it. So the list names the extension
// point rather than the reason, and the reason lives in testkit's own docs.
func (c *Contract) Unchecked() []Coverage {
	out := make([]Coverage, 0, len(c.Coverage))
	for _, cov := range c.Coverage {
		if !cov.Checked {
			out = append(out, cov)
		}
	}
	return out
}

// Unwritten is what this file does not check and a consumer could.
func (c *Contract) Unwritten() []Coverage {
	out := make([]Coverage, 0, len(c.Coverage))
	for _, cov := range c.Unchecked() {
		if !cov.Elsewhere() {
			out = append(out, cov)
		}
	}
	return out
}

// Elsewhere is what this file does not check and a consumer should not.
//
// Split from [Contract.Unwritten] because the advice differs and getting it
// wrong is worse than saying nothing: a header telling somebody to hand-write
// `deleteremoves` is telling them to state a property that needs a reference
// implementation, against a run that has none.
func (c *Contract) Elsewhere() []Coverage {
	out := make([]Coverage, 0, len(c.Coverage))
	for _, cov := range c.Unchecked() {
		if cov.Elsewhere() {
			out = append(out, cov)
		}
	}
	return out
}

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
		contract := &Contract{
			BaseEmit:      sdk.EmitBase(c, iface),
			Subject:       subjectOf(iface),
			EntryName:     "Assert" + iface.Name + "Contract",
			Fixture:       fixture,
			Seed:          seedOf(fixture, methods),
			Double:        doubleOf(doubles, iface),
			Methods:       methods,
			Coverage:      coverageOf(methods, modelWillRun(iface)),
			Unfalsifiable: unfalsifiableReason(iface, doubles),
		}
		queued := []sdk.EmitNode{contract}
		if f, provable := falsificationOf(ctx, c, iface, contract); provable {
			// The companion output, which drives every check against a stand-in
			// that violates it. Queued beside the harness rather than instead of
			// it: the two are routed apart by tag, and a run that produced a
			// harness it could not prove still owes the harness.
			queued = append(queued, f)
		}
		if err := sdk.QueueEmit(ctx.Store.Emit(), c, SlotName, iface, queued...); err != nil {
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
		roles, partners := contractDataOf(bag)
		out = append(out, Method{
			Sig:              golang.SigOf(src),
			CheckType:        iface.Name + src.Name + "Check",
			Mixins:           shape.Mixins(bag),
			IntegrationOnly:  slices.Contains(shape.Mixins(bag), MixinIntegrationOnly),
			Contracts:        shape.Contracts(bag),
			mixinParams:      mixinParamsOf(bag),
			contractRoles:    roles,
			contractPartners: partners,
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
		{MixinPartition, MixinPartitionRead},
		{MixinPartition, MixinPartitionAxis},
		{MixinHooks, MixinHooksParam},
		{MixinSample, MixinSampleParam},
		{MixinAfterClose, MixinAfterCloseClose},
		{MixinAfterClose, MixinAfterCloseSentinel},
		{MixinValidates, MixinValidatesParam},
		{MixinWrappedVia, MixinWrappedViaParam},
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
	if m.ReturnsError() && len(m.ValueReturns()) > 0 && m.HasInput() &&
		!slices.Contains(m.Mixins, "total") {
		// The only one whose meaning is in the value: a miss check called with
		// the value the subject was seeded with succeeds, and asserts nothing.
		//
		// HasInput because the check reaches the failure it is about through the
		// alternate value, and a method taking nothing after its context leaves
		// nowhere to put one; the check skips visibly where even the alternate
		// succeeds. The total mixin is the declared form of that totality — a
		// claim that no input fails — so nothing is emitted against it rather
		// than a check that skips by construction.
		out = append(out, base(KindZeroOnError, "an error carries the zero value", "ZeroOnError", other, true))
	}
	return out
}

// batchSizeCheck builds "a batch read answers once per key requested".
//
// The one detector claim that is about arity rather than absence, and the one
// the miss family cannot state: a batch reader answering once for many keys
// returns a plausible non-empty slice, so every zero comparison passes.
//
// Two keys, which is what makes it able to fail at all — the derived pair is
// exactly the second element a variadic call otherwise never gets. A method
// whose variadic parameter has no alternate is one the check cannot vary, so
// nothing is generated rather than a call with one element and an assertion
// that one came back.
func batchSizeCheck(
	c *sdk.Provenance, iface *sdk.Interface, f Fixture, m Method,
) (*Check, bool) {
	if shape.Get(m.Source.Meta()) != batchreader.Name {
		return nil, false
	}
	v := m.VariadicParam()
	if v == nil || len(m.CallArgs()) != 1 || len(m.ValueReturns()) != 1 {
		return nil, false
	}
	field, found := f.Field(f.FieldFor(*v))
	if !found || !field.OK() {
		return nil, false
	}

	ck := &Check{
		BaseEmit:          sdk.EmitBase(c, iface),
		Subject:           subjectOf(iface),
		KindName:          KindBatchSize,
		Subtest:           "answers once per key",
		Func:              "Assert" + iface.Name + m.Name + "AnswersPerKey",
		Path:              m.Name + "/answers once per key",
		Args:              fixtureArgs(f, m, false),
		NeedsDerivedInput: true,
		Method:            m,
	}
	ck.Extra = []ExtraArg{{
		Name:  OtherIdent(v.Name),
		Field: field.OtherName(),
		Type:  v.Type,
	}}
	ck.SecondCall = []string{v.Name, OtherIdent(v.Name)}
	return ck, true
}

// The mixins this generator generates a check for, and the parameters it reads.
//
// Named here rather than taken from eidos's own constants at each use so the
// set a reader has to hold is one list. Each is suite-owned under
// docs/adr/0018 — no [engine/model/law] property covers it — and each is
// derivable from the stamp plus the signature, which is what excludes the rest:
// `validates` and `scope` need a value no run can invent, and `sideeffect`,
// `partition`, `hooks` and `sample` name a partner the mixin schema declares
// no parameter for.
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
	MixinPartition       = partition.Name
	MixinPartitionRead   = partition.ParamRead
	MixinPartitionAxis   = partition.ParamAxis
	MixinHooks           = hooks.Name
	MixinHooksParam      = hooks.ParamRegister
	MixinSample          = sample.Name
	MixinSampleParam     = sample.ParamBuilder
	MixinValidates       = validates.Name
	MixinValidatesParam  = validates.ParamFn
	MixinWrappedVia      = wrappedvia.Name
	MixinWrappedViaParam = wrappedvia.ParamFn

	MixinIdempotent         = idempotent.Name
	MixinConcurrent         = concurrent.Name
	MixinAfterClose         = lifecycleafterclose.Name
	MixinAfterCloseClose    = lifecycleafterclose.ParamClose
	MixinAfterCloseSentinel = lifecycleafterclose.ParamSentinel
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
	if ck, ok := batchSizeCheck(c, iface, f, m); ok {
		return []*Check{ck}
	}
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
// unwritten: `validates` and `scope` need a value no run can invent, four name
// a partner the mixin schema declares no parameter for, `concurrent` and
// `concurrentreaders` assert nothing without `-race`, `retrysucceeds` has no
// attempt count to read, `integrationonly` is a build tag rather than an
// assertion, and `deprecated` is a fact about a method rather than a claim about
// its behaviour — it is stated in the generated documentation instead.
// agreementCheck builds "the method refuses what the validator refuses".
//
// Agreement rather than rejection, for the reason if-match's is: "what the
// validator rejects, the method rejects" needs a value the validator rejects,
// and nothing in the directive says which one that is. What both halves can
// always be asked is whether they agree on the value the run has.
//
// The validator has to answer about the very value the method takes, and to
// report its verdict as an error — a validator returning something else is a
// method the directive happened to name.
func agreementCheck(f Fixture, m, fn Method, base checkBuilder) (*Check, bool) {
	if !m.ReturnsError() || !fn.ReturnsError() || len(fn.ValueReturns()) > 0 {
		return nil, false
	}
	if !sameArgs(m, fn) {
		return nil, false
	}
	args, spellable := partnerArgs(f, m, fn)
	if !spellable {
		return nil, false
	}

	ck := base(KindValidates, MixinValidates, "AgreesWith"+fn.Name, fixtureArgs(f, m, false))
	ck.Partner, ck.PartnerArgs = &fn, args
	return ck, true
}

// wrappingCheck builds "the failure carries the cause the mixin names".
//
// Conditional on the cause being one: a subject with nothing wrong has no
// wrapped error to show, and demanding one would fail an implementation that is
// simply healthy. So the check asks the cause first and asserts only when there
// is something to wrap — which is what makes it able to fail without being able
// to fail wrongly.
func wrappingCheck(f Fixture, m, cause Method, base checkBuilder) (*Check, bool) {
	if !m.ReturnsError() || !cause.ReturnsError() || len(cause.ValueReturns()) > 0 {
		return nil, false
	}
	args, spellable := partnerArgs(f, m, cause)
	if !spellable {
		return nil, false
	}

	ck := base(KindWrappedVia, MixinWrappedVia, "Wraps"+cause.Name, fixtureArgs(f, m, false))
	ck.Partner, ck.PartnerArgs = &cause, args
	return ck, true
}

// sameArgs reports whether two methods take the same parameters after their
// contexts.
//
// Both the types and the fixture fields, because a check receives the method's
// own arguments and calls the partner with them: two parameters at one type
// under different names resolve to different fields, and one of them is not in
// scope.
func sameArgs(m, other Method) bool {
	a, b := m.CallArgs(), other.CallArgs()
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if !a[i].Source.Equal(b[i].Source) {
			return false
		}
	}
	return true
}

// builds reports whether the partner produces a value the method accepts.
//
// Checked rather than assumed, because the check's whole content is passing one
// to the other: a builder returning something the method does not take gives a
// call that will not compile, and a render error is a file that came out short.
//
// One parameter and one produced value. With several of either, which feeds
// which is a guess — the mixin names the builder and says nothing about the
// slot, and that is exactly the ambiguity `partition` needed an axis to settle.
func builds(m, builder Method) bool {
	args, produced := m.CallArgs(), builder.ValueReturns()
	if len(args) != 1 || len(produced) != 1 {
		return false
	}
	return args[0].Source.Equal(produced[0].Source)
}

// callbackParam returns the func-typed parameter a registration partner takes.
//
// Exactly one, and it must be a func: `OnEvent(fn func(string))` is a
// registration, `OnEvent(name string)` is something else the annotator happened
// to be pointed at. A partner that is not a registration generates no check
// rather than a literal the toolchain refuses.
func callbackParam(partner Method) (*CallbackSig, bool) {
	args := partner.CallArgs()
	if len(args) != 1 {
		return nil, false
	}
	src := args[0].Source
	if src == nil || src.TypeKind != sdk.TypeRefFunc {
		return nil, false
	}

	sig := &CallbackSig{Name: args[0].Name}
	for _, p := range src.FuncParams {
		sig.Params = append(sig.Params, golang.FromNode(p))
	}
	for _, r := range src.FuncReturns {
		sig.Returns = append(sig.Returns, golang.FromNode(r))
	}
	return sig, true
}

// partitionCheck builds the isolation check, or reports that this method cannot
// state it.
//
// Isolation is a claim about two writes not reaching each other, so the check
// writes the same everything under two values of one parameter and reads one
// back. Which parameter that is has to be said: `Put(ctx, partition, key, v)`
// and `Read(ctx, partition, key)` share both, the types are identical, and
// nothing else distinguishes the axis from the key.
//
// An earlier version varied every parameter instead, which is why the axis is
// worth insisting on — the two writes then never collided on a key, and the
// check passed against an implementation ignoring partitions entirely. A check
// that cannot fail is worse than no check, so a method naming no axis generates
// nothing rather than something that looks like coverage.
func partitionCheck(f Fixture, m, read Method, base checkBuilder) (*Check, bool) {
	axis, named := m.MixinParam(MixinPartition, MixinPartitionAxis)
	if !named {
		return nil, false
	}
	args, spellable := partnerArgs(f, m, read)
	if !spellable {
		return nil, false
	}

	// One extra per parameter, but only the axis takes its alternate: holding
	// the key fixed is what makes the two writes collide, which is the whole
	// premise. eidos validates that the axis names a parameter, so a miss here
	// is a method the annotator never saw.
	var (
		extra   []ExtraArg
		second  = make([]string, 0, len(m.CallArgs()))
		payload string
		isAxis  bool
	)
	// Which parameters the reader shares is what separates an identifier from a
	// payload: Read takes the partition and the key, so `value` is what Put
	// carries rather than what it addresses.
	shared := map[string]bool{}
	for _, p := range read.CallArgs() {
		shared[f.FieldFor(p)] = true
	}

	for _, p := range m.CallArgs() {
		isTheAxis := p.Name == axis
		// Hold every identifier so the two writes land on the same slot, and
		// vary every payload so an overwrite is visible. Holding the payload
		// too was the second near-miss: a subject ignoring partitions clobbers
		// the first write with an identical value, and the read still returns
		// what the check expects.
		if !isTheAxis && shared[f.FieldFor(p)] {
			second = append(second, p.Name)
			continue
		}
		field, found := f.Field(f.FieldFor(p))
		if !found || !field.OK() {
			// No second value is no second write worth making.
			return nil, false
		}
		ident := OtherIdent(p.Name)
		extra = append(extra, ExtraArg{Name: ident, Field: field.OtherName(), Type: p.Type})
		second = append(second, ident)
		if isTheAxis {
			isAxis = true
			continue
		}
		// A parameter that is neither the axis nor an identifier is the
		// payload, and the read is held up to the first write's copy of it:
		// what Read must return, rather than what it must not. Both are the
		// same claim, and this one also fails a subject returning nothing.
		payload = p.Name
	}
	if !isAxis {
		return nil, false
	}

	if payload == "" {
		// Every parameter identifies the slot, so the two writes differ in
		// where they land and in nothing else — there is no value for the read
		// to be wrong about.
		return nil, false
	}

	ck := base(KindPartition, MixinPartition, "IsolatesPartitions", fixtureArgs(f, m, false))
	ck.Extra, ck.SecondCall, ck.CompareAgainst = extra, second, payload
	ck.Partner, ck.PartnerArgs = &read, args
	return ck, true
}

// OtherIdent names the identifier a check binds a field's alternate to.
//
// Derived from the parameter so a reader meets `partitionOther` beside
// `partition` and does not have to look up which of two values is which.
func OtherIdent(name string) string { return name + OtherSuffix }

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
// teardownShaped reports the one signature "a second call answers the same"
// can be stated against without a value: context in, error out, nothing else.
func teardownShaped(m Method) bool {
	return m.TakesContext() && m.ReturnsError() &&
		len(m.ValueReturns()) == 0 && !m.HasInput()
}

// qualifiedExpr lifts a resolver-qualified symbol into an expression, false
// for a bare name with no package to import it from.
func qualifiedExpr(v string) (*sdk.Expr, bool) {
	i := strings.LastIndexByte(v, '.')
	if i <= 0 || i == len(v)-1 {
		return nil, false
	}
	return sdk.NewExternal(v[:i], v[i+1:]), true
}

// spellableBuilder reports whether every argument the builder takes has a
// fixture field to draw from.
func spellableBuilder(f Fixture, p Method) bool {
	for _, arg := range p.CallArgs() {
		field, held := f.Field(f.FieldFor(arg))
		if !held || !field.OK() {
			return false
		}
	}
	return true
}

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
	return methodNamed(methods, m.MixinPartner(mixin, param))
}

// checkBuilder composes the fields every classification-derived check shares,
// leaving each selector to set only what its own claim needs.
//
// A named type rather than a closure literal at each call site: two families
// select checks the same way and a third would have been a third copy of the
// same seven fields, which is where two of them drift apart.
type checkBuilder func(kind sdk.Kind, subtest, suffix string, args []string) *Check

// checkFor binds a builder to one method of one interface.
func checkFor(c *sdk.Provenance, iface *sdk.Interface, m Method) checkBuilder {
	return func(kind sdk.Kind, subtest, suffix string, args []string) *Check {
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
}

func mixinChecks(
	c *sdk.Provenance, iface *sdk.Interface, f Fixture, m Method, methods []Method,
) []*Check {
	base := checkFor(c, iface, m)

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
	if p := partnerOf(methods, m, MixinSample, MixinSampleParam); p != nil && builds(m, *p) {
		// The sampled value stays the builder's answer — handing the method a
		// fixture value instead would test the derivation rather than the
		// pair. The builder's own arguments are the fixture's to supply,
		// though: a builder taking anything after its context used to render
		// a call with those arguments missing, latent until a source declared
		// one. A builder whose arguments nothing can spell drops the check,
		// with the header saying so.
		if spellableBuilder(f, *p) {
			ck := base(KindSample, MixinSample, "AcceptsASampledInput", fixtureArgs(f, *p, false))
			ck.Partner = p
			for _, arg := range p.CallArgs() {
				ck.PartnerArgs = append(ck.PartnerArgs, arg.Name)
			}
			out = append(out, ck)
		}
	}
	if p := partnerOf(methods, m, MixinHooks, MixinHooksParam); p != nil {
		if cb, ok := callbackParam(*p); ok {
			ck := base(KindHooks, MixinHooks, "FiresRegisteredHooks", fixtureArgs(f, m, false))
			ck.Partner, ck.Callback = p, cb
			out = append(out, ck)
		}
	}
	if slices.Contains(m.Mixins, MixinIdempotent) && teardownShaped(m) {
		// The declared claim, on the one shape where "again" needs no value:
		// a second teardown answers what the first did. Gated on the mixin —
		// a bare lifecycle shape makes no such promise, and os.File-style
		// subjects legitimately refuse the second call.
		out = append(out, base(KindCloseIdempotent, MixinIdempotent, "CloseTwice", nil))
	}
	if slices.Contains(m.Mixins, MixinConcurrent) {
		// Four callers under the race detector: the mixin's whole claim is
		// that parallel use is safe, and no other generated file so much as
		// starts a goroutine.
		out = append(out, base(KindConcurrentSmoke, MixinConcurrent, "SurvivesConcurrentCallers",
			fixtureArgs(f, m, false)))
	}
	if p := partnerOf(methods, m, MixinAfterClose, MixinAfterCloseClose); p != nil {
		// The op is the carrier and the close its partner; the sentinel is
		// the declaration's own, because "refused" without an identity lets
		// any unrelated failure pass as closure discipline.
		if sym, stamped := m.MixinParam(MixinAfterClose, MixinAfterCloseSentinel); stamped {
			if ref, qualified := qualifiedExpr(sym); qualified {
				ck := base(KindUseAfterClose, MixinAfterClose, "RefusesAfterClose",
					fixtureArgs(f, m, false))
				ck.Partner, ck.Sentinel = p, ref
				out = append(out, ck)
			}
		}
	}
	if p := partnerOf(methods, m, MixinPartition, MixinPartitionRead); p != nil {
		if ck, ok := partitionCheck(f, m, *p, base); ok {
			out = append(out, ck)
		}
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
	if p := partnerOf(methods, m, MixinValidates, MixinValidatesParam); p != nil {
		if ck, ok := agreementCheck(f, m, *p, base); ok {
			out = append(out, ck)
		}
	}
	if p := partnerOf(methods, m, MixinWrappedVia, MixinWrappedViaParam); p != nil {
		if ck, ok := wrappingCheck(f, m, *p, base); ok {
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
		Witnesses:      d.Witnesses,
	}
}

// ModelDirective is the model generator's directive name, respelled here
// because that generator imports this one and Go permits no import back.
// Exported so the conformance gate can hold the two spellings equal.
const ModelDirective = "model"

// ModelWitnessKey is the model directive's witness key, respelled here for
// the same reason [ModelDirective] is: the suite cannot import the generator
// that imports it, and the header's "will the model tier run" predicate has
// to ask the same question the model generator answers.
const ModelWitnessKey = "witness"

// modelWillRun reports whether the model tier will actually emit for this
// interface: armed by its directive, and — where the interface is generic —
// carrying the witness list that names the types the property runs at. The
// model generator refuses a generic interface without one, and a header
// pointing at output that will not exist is the lie this predicate stops.
func modelWillRun(iface *sdk.Interface) bool {
	if !iface.HasPositiveDirective(ModelDirective) {
		return false
	}
	if len(iface.TypeParams) == 0 {
		return true
	}
	dir := iface.Directive(ModelDirective)
	if dir == nil {
		return false
	}
	_, witnessed := dir.KV[ModelWitnessKey]
	return witnessed
}

// subjectOf names the interface every emit value for it is about.
func subjectOf(iface *sdk.Interface) Subject {
	return Subject{
		IfaceName:      iface.Name,
		IfaceRef:       golang.RefFor(iface.Name, iface.Package),
		Runtime:        Module,
		IntegrationEnv: GoIntegrationEnv,
		ClockRef:       golang.RefFor("Clock", Module+"/clock"),
		TypeParams:     golang.TypeParamDecls(iface.TypeParams),
		TypeArgs:       golang.TypeParamNames(iface.TypeParams),
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
