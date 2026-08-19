// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package suite

import (
	"fmt"
	"slices"

	"go.thesmos.sh/eidos/lang/golang"
	"go.thesmos.sh/eidos/plugins/annotator/shape"
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
	"go.thesmos.sh/eidos/plugins/annotator/shape/mixins/total"
	"go.thesmos.sh/eidos/plugins/annotator/shape/mixins/ttl"
	"go.thesmos.sh/eidos/plugins/annotator/shape/mixins/validates"
	"go.thesmos.sh/eidos/plugins/annotator/shape/mixins/wrappedvia"
	"go.thesmos.sh/eidos/sdk"
	sdkgolang "go.thesmos.sh/eidos/sdk/golang"

	"go.thesmos.sh/testkit/generator/suite/projection"
)

// Name is the plugin's stable identifier.
const Name = "suite"

// Capability is the label the plugin advertises, so the generators that read
// this one's projection — bench, fuzz, model — can declare the dependency.
const Capability = "suite"

// Version composes into the pipeline's plugin fingerprint. Bump it on any
// change to what this plugin emits, the projection or the templates alike.
const Version = "1.16.0"

// DirectiveName is the bare directive name — without the `//testkit:` prefix —
// that opts an interface in.
const DirectiveName sdk.DirectiveName = "suite"

// SlotName is the [sdk.EmitFile] slot the harness lands in.
const SlotName = "top"

// KindContract is the emit kind for the harness as a whole. The backend
// resolves a template by the kind's string value, so the constant doubles as
// the template's name.
const KindContract sdk.Kind = "suite.contract"

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
		// The rewrite's body templates share this FS, so their
		// functions join the merged map the backend parses it with.
		Funcs(renderFuncs()).
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
			// The Describe text says it takes no arguments; this is what makes
			// that true. An empty AllowedKeys means unrestricted, so before
			// DenyKeys existed a stray key parsed, validated and stamped
			// nothing, and whoever wrote it believed it had an effect.
			DenyKeys().
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

// TakesContext reports whether the method's first parameter is a context.
//
// The gate on three of the five signature-derived checks: cancellation, an
// expired deadline and a nil context are all claims about a parameter a method
// may not take, and emitting them for one that does not would not compile.
func (m Method) TakesContext() bool {
	return len(m.Params) > 0 && golang.IsContext(m.Params[0].Source)
}

// Shape returns the detector the annotator stamped on this method, empty when
// it stamped none.
func (m Method) Shape() string {
	if m.Source == nil {
		return ""
	}
	return shape.Get(m.Source.Meta())
}

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

	// Unseeded says why no seed was derived, empty where one was.
	//
	// A harness that seeds nothing runs every read check against a fresh
	// subject, where a miss and a bug are the same observation. The header
	// carries the reason so the reader knows which of the three exits was
	// taken and therefore what would close it.
	Unseeded string

	Methods []Method

	// Token qualifies every identifier the file emits, so the templates
	// compose names from one word rather than each lower-casing the
	// interface for itself.
	Token string

	// Inventory is every check the derivers licensed, and Index the
	// typed surface a consumer drops one through. Both are projections
	// of the same nodes, which is what keeps the index from naming a
	// check the run does not emit.
	Inventory projection.Inventory
	Index     projection.IndexPlan

	// Refusals are the checks the rules reached and could not derive.
	// They render into the header: a claim the reader cannot see
	// refused reads as a claim this file checks.
	Refusals []Refusal
}

// Kind returns [KindContract].
func (*Contract) Kind() sdk.Kind { return KindContract }

// Generate queues one harness per interface carrying the directive.
func (*Plugin) Generate(ctx *sdk.GeneratorContext) error {
	c := sdk.NewProvenance(Name)
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
		// The rewrite's transition state: the projection carrier still
		// queues — the model tier reads Methods, Fixture and Subject
		// from it — and the check emission is the deriver inventory's,
		// rendered once the body and structural templates land. The
		// incumbent's check assembly is deleted, not dormant.
		methods := methodsOf(iface, set)
		fixture := fixtureOf(ctx, iface, methods)
		for i := range methods {
			// The fixture-field correspondence the derivers draw
			// through — populated here since the incumbent's check
			// assembly, its previous home, is gone.
			methods[i].ArgFields = fixtureArgs(fixture, methods[i], false)
		}
		seed, unseeded := seedOf(fixture, methods)

		token := projection.Token(iface.Name)
		inventory, refusals := InventoryOf(Iface{
			Name:      iface.Name,
			Token:     token,
			Qualifier: projection.IDQualifier(iface.Name),
			Methods:   methods,
			Fixture:   fixture,
		})
		if err := inventory.Verify(); err != nil {
			// The run's own invariants, held before anything renders. A
			// deriver bug caught here names the check it is about; the
			// same bug reaching a consumer is a compile error in a file
			// they did not write.
			ctx.Diag.Errorf(iface.Pos(), "%s: %s: %v", Name, iface.Name, err)
			continue
		}
		index, err := projection.IndexOf(inventory)
		if err != nil {
			ctx.Diag.Errorf(iface.Pos(), "%s: %s: %v", Name, iface.Name, err)
			continue
		}

		contract := &Contract{
			BaseEmit:  sdk.EmitBase(c, iface),
			Subject:   subjectOf(iface),
			EntryName: "Assert" + iface.Name + "Contract",
			Fixture:   fixture,
			Seed:      seed,
			Unseeded:  unseeded,
			Methods:   methods,
			Token:     token,
			Inventory: inventory,
			Index:     index,
			Refusals:  refusals,
		}
		if unseeded != "" {
			// A harness that seeds nothing runs every read check against a
			// fresh subject, where a miss and a bug look identical. A
			// warning, because the consumer's own seed closes it.
			ctx.Diag.Warnf(iface.Pos(),
				"%s: %s derives no seed — %s; supply one with %sSeed",
				Name, iface.Name, unseeded, iface.Name)
		}
		if err := sdk.QueueEmit(ctx.Store.Emit(), c, SlotName, iface, contract); err != nil {
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
		{MixinTTL, MixinTTLNotFound},
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

// The mixins this generator generates a check for, and the parameters it reads.
//
// Named here rather than taken from eidos's own constants at each use so the
// set a reader has to hold is one list. Each is suite-owned under
// docs/adr/0018 — no [engine/model/law] property covers it — and each is
// derivable from the stamp plus the signature, which is what excludes the rest:
// `validates` and `scope` need a value no run can invent, and `sideeffect`,
// `partition`, `hooks` and `sample` name a partner the mixin schema declares
// no parameter for.
//
// The one exception is `ttl`, whose law is the model tier's: its row exists
// because the reader-miss claim speaks the sentinel the ttl declaration
// names, and wording reads the declared home rather than respelling it.
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
	MixinTTL                = ttl.Name
	MixinTTLNotFound        = ttl.ParamNotFound

	// MixinTotal is read as an exclusion, not a check: totality is the
	// declared claim that no input fails, so the zero-on-error family
	// is not emitted against it. The law half is the model tier's.
	MixinTotal = total.Name
)

// partnerArgs names the identifiers a call to the partner is handed, and says
// why it could not be spelled where it could not.
//
// A generated check receives the *annotated* method's parameters and nothing
// else, so the partner can only be called with values already in scope. Both
// resolve their parameters to fixture fields, and a field they share is one
// identifier serving both — which is the ordinary case, since a partner
// observing an effect keyed on something observes it by the same key.
//
// # The correspondence has to be derivable, not guessed
//
// The rule the `partition` mixin states with `axis=`, generalised. That mixin
// makes eidos reject an axis the partner does not spell identically, because
// `Put(ctx, partition, key, v)` and `Read(ctx, partition, key)` have two
// parameters of one type and nothing else distinguishes them — a generator
// matching by position would silently write a check about the wrong one.
//
// Identical spelling is one way to be unambiguous. Being the only parameter of
// that type is another, and it is the shape the corpus kept losing: a partner
// declaring `k` where the method declares `key`, at one `string` each, is not
// ambiguous by any reading, and matching by identifier alone declined it.
//
// So the correspondence is derived in two passes — same fixture field first,
// then sole remaining parameter of the type — and anything the two passes
// leave undecided is reported rather than dropped. A check that does not exist
// and a classification that says nothing about why look identical from the
// output, and the whole tier this sits in is about that resemblance.
//
// The widening never reaches across types: a parameter is spelled from one of
// the method's own, never invented, so a shape the passes decline is one where
// a check would have had a signature no other check has.
// teardownShaped reports the one signature "a second call answers the same"
// can be stated against without a value: context in, error out, nothing else.
func teardownShaped(m Method) bool {
	return m.TakesContext() && m.ReturnsError() &&
		len(m.ValueReturns()) == 0 && !m.HasInput()
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
