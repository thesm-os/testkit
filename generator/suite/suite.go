// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package suite

import (
	"fmt"

	"go.thesmos.sh/eidos/lang/golang"
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

	// Args names the fixture fields this check is handed, one per parameter the
	// method takes after its context.
	//
	// Field names rather than values, because the fixture is one struct a
	// consumer may replace whole: a check holding a literal would keep running
	// against the derived value after an override replaced it.
	Args []string

	// Method is the signature under check.
	Method Method
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

	// Checks are the generated assertions for this method, in the order the
	// entry point runs them.
	Checks []*Check
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
		methods := methodsOf(c, iface, set)
		fixture := fixtureOf(ctx, iface, methods)
		methods = withDerivableChecks(ctx, iface, fixture, methods)
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
func methodsOf(c *sdk.Provenance, iface *sdk.Interface, set sdk.MethodSetResult) []Method {
	out := make([]Method, 0, len(set.Methods))
	for _, src := range set.Methods {
		m := Method{
			Sig:       golang.SigOf(src),
			CheckType: iface.Name + src.Name + "Check",
		}
		m.Checks = signatureChecks(c, iface, m)
		out = append(out, m)
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
func signatureChecks(c *sdk.Provenance, iface *sdk.Interface, m Method) []*Check {
	// The sample values for the ordinary checks, and the second set for the one
	// that has to miss. A zero-value check called with the value the subject was
	// seeded with succeeds, and then asserts nothing.
	args := fixtureArgs(m, false)
	other := fixtureArgs(m, true)

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
	if m.ReturnsError() && len(m.ValueReturns()) > 0 {
		// The only one whose meaning is in the value: a miss check called with
		// the value the subject was seeded with succeeds, and asserts nothing.
		out = append(out, base(KindZeroOnError, "an error carries the zero value", "ZeroOnError", other, true))
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
		IfaceName: iface.Name,
		IfaceRef:  golang.RefFor(iface.Name, iface.Package),
		Runtime:   Module,
	}
}

// fixtureArgs names the fixture field per parameter the method takes after its
// context, taking the second value of each when alternate is set.
func fixtureArgs(m Method, alternate bool) []string {
	args := m.CallArgs()
	out := make([]string, 0, len(args))
	for _, p := range args {
		if alternate {
			out = append(out, p.Field+OtherSuffix)
			continue
		}
		out = append(out, p.Field)
	}
	return out
}
