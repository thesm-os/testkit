// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package suite

import (
	"strings"

	"go.thesmos.sh/eidos/lang/golang"
	"go.thesmos.sh/eidos/sdk"

	"go.thesmos.sh/testkit/generator/stub"
)

// KindFalsification is the emit kind for the companion output as a whole.
const KindFalsification sdk.Kind = "suite.falsification"

// The emit kinds for the per-check violators.
//
// One per check kind, and named to mirror it: `suite.check.cancel` is proved by
// `suite.violate.cancel`. The symmetry is the point — a check kind added
// without a violator is a check nobody proved, and the pairing is what makes
// that visible in the directory listing rather than only in the output.
const (
	KindViolateSmoke       sdk.Kind = "suite.violate.smoke"
	KindViolateCancel      sdk.Kind = "suite.violate.cancel"
	KindViolateDeadline    sdk.Kind = "suite.violate.deadline"
	KindViolateNilContext  sdk.Kind = "suite.violate.nilcontext"
	KindViolateZeroOnError sdk.Kind = "suite.violate.zeroonerror"
	KindViolateIfAbsent    sdk.Kind = "suite.violate.ifabsent"
	KindViolatePartition   sdk.Kind = "suite.violate.partition"
	KindViolateOrderAfter  sdk.Kind = "suite.violate.orderafter"
	KindViolateSideEffect  sdk.Kind = "suite.violate.sideeffect"
	KindViolateHooks       sdk.Kind = "suite.violate.hooks"
	KindViolateNilSafe     sdk.Kind = "suite.violate.nilsafe"
	KindViolateSample      sdk.Kind = "suite.violate.sample"
	KindViolateValidates   sdk.Kind = "suite.violate.validates"
	KindViolateWrappedVia  sdk.Kind = "suite.violate.wrappedvia"
	KindViolateBatchSize   sdk.Kind = "suite.violate.batchsize"
	KindViolateOutbox      sdk.Kind = "suite.violate.outbox"
	KindViolateIfMatch     sdk.Kind = "suite.violate.ifmatch"
	KindViolateMissZero    sdk.Kind = "suite.violate.misszero"
	KindViolateMissFlag    sdk.Kind = "suite.violate.missflag"
	KindViolateTimeout     sdk.Kind = "suite.violate.timeout"

	KindViolateAnswerRoundTrip sdk.Kind = "suite.violate.answerroundtrip"

	KindViolateCleanTeardown   sdk.Kind = "suite.violate.cleanteardown"
	KindViolateCloseIdempotent sdk.Kind = "suite.violate.closeidempotent"
	KindViolateUseAfterClose   sdk.Kind = "suite.violate.useafterclose"
	KindViolateConcurrentSmoke sdk.Kind = "suite.violate.concurrentsmoke"
)

// Falsification is the emit value rendered into the tagged test output.
//
// Every generated check driven against a stand-in that violates it, so a check
// that cannot fail is a build failure rather than a quiet line of coverage.
//
// # Why a separate file rather than an exported entry point
//
// A `Test` function outside a `_test.go` file never runs, so the proof has to
// live here to execute at all. Beyond that, the suffix earns the
// external-test-package shift from Layout: the guard reaches the harness across
// a package boundary the way a consumer does, which also demonstrates that the
// exported surface is sufficient to drive a check from outside.
//
// Keeping it out of [Contract] keeps two different failures answerable. When
// the entry point fails, a consumer needs to know whether their implementation
// is wrong or the generated harness is, and those are opposite actions.
type Falsification struct {
	sdk.BaseEmit
	Subject

	// Double is the generated stand-in every violator configures. Nil is
	// impossible here: [falsificationOf] declines to queue anything without
	// one, since a violator is a stand-in behaving badly and there is nothing
	// else to configure.
	Double *Double

	// FixtureCtor names the derived-input constructor in the harness, so a
	// guard hands the check the same values the run does.
	FixtureCtor string

	// Witnesses are the concrete types every guard instantiates the generic
	// harness at — the double's own, so the two companions prove one
	// instantiation. Empty for a non-generic interface.
	Witnesses []sdk.Ref

	// Cases are the checks with a derivable violator, in harness order.
	Cases []*Violation

	// Unproven names the checks with none, and why.
	//
	// Named rather than omitted, for the reason a generated file names an
	// uncovered classification: a file silent about a check is
	// indistinguishable from one where the generator failed to handle it.
	Unproven []Unproven

	// HarnessPkg is the package the primary output landed in, which is where
	// every symbol a guard names lives.
	//
	// Not known during Generate — Layout decides it, and `out=` can move the
	// file afterwards — so it is set provisionally and corrected in
	// [Falsification.SetOutputPackages]. A wrong package is a compile error
	// naming the symbol; a bare name would silently bind to something else.
	HarnessPkg string
}

// Kind returns [KindFalsification].
func (*Falsification) Kind() sdk.Kind { return KindFalsification }

// SetOutputPackages repoints every guard at wherever Layout put the harness.
//
// Called at most once per value, after all Targets resolve, and the map may be
// partial: a run that recorded routing errors reaches dispatch with some tags
// missing, and the primary's entry can be present-but-empty. Indexed
// defensively for exactly that.
func (f *Falsification) SetOutputPackages(byTag map[string]string) {
	path := byTag[""]
	if path == "" {
		return
	}
	f.HarnessPkg = path
	for _, c := range f.Cases {
		c.HarnessPkg = path
	}
}

// witnessedMethod rewrites a method's signature at the witnesses: every
// parameter and return naming a type parameter lands at the concrete type a
// guard can spell. The projection is shared with the harness's own rendering,
// so the rewrite clones, and a nil binding map returns the method untouched.
func witnessedMethod(m Method, by map[string]sdk.Ref) Method {
	if by == nil {
		return m
	}
	sig := *m.Sig
	params := make([]golang.Param, len(sig.Params))
	for i, p := range sig.Params {
		p.Type = golang.SubstituteTypeParams(p.Type, by)
		params[i] = p
	}
	returns := make([]golang.Return, len(sig.Returns))
	for i, r := range sig.Returns {
		r.Type = golang.SubstituteTypeParams(r.Type, by)
		returns[i] = r
	}
	sig.Params, sig.Returns = params, returns
	m.Sig = &sig
	return m
}

// witnessedPartner is [witnessedMethod] over the nilable partner slot.
func witnessedPartner(m *Method, by map[string]sdk.Ref) *Method {
	if m == nil || by == nil {
		return m
	}
	w := witnessedMethod(*m, by)
	return &w
}

// unfalsifiableReason says why this interface gets no companion output, empty
// where it gets one.
//
// Two reasons, and both are facts about what a guard can be rather than about
// the checks. A guard configures the generated stand-in, so an interface with
// none has nothing to make behave badly. And a guard is a `Test` function,
// which cannot carry type arguments — so a generic interface would need a
// concrete instantiation, and nothing in the source says which.
func unfalsifiableReason(iface *sdk.Interface, doubles map[sdk.Node]*stub.Stub) string {
	double, hosted := doubles[sdk.Node(iface)]
	if !hosted {
		return "the interface declares no //testkit:stub, so there is nothing to break"
	}
	if len(iface.TypeParams) > 0 && len(double.Witnesses) == 0 {
		return "the interface is generic and its constraints derive no witnesses; " +
			"name the types to prove it with through witness= on //testkit:stub"
	}
	return ""
}

// CheckKinds returns every kind of check this generator emits.
//
// Enumerated so the falsification pass can be held to it: a kind added without
// a stand-in that breaks it is a check nobody proved, and the whole companion
// output exists to make that visible.
func CheckKinds() []sdk.Kind {
	return []sdk.Kind{
		KindSmoke, KindCancel, KindDeadline, KindNilContext, KindZeroOnError,
		KindMissZero, KindMissFlag, KindBatchSize, KindAnswerRoundTrip,
		KindNilSafe, KindTimeout, KindOrderAfter, KindSideEffect, KindPartition,
		KindHooks, KindSample, KindValidates, KindWrappedVia,
		KindIfAbsent, KindIfMatch, KindOutbox,
		KindCleanTeardown, KindCloseIdempotent, KindUseAfterClose, KindConcurrentSmoke,
	}
}

// ViolatorFor returns the emit kind of the stand-in that breaks a check, and
// whether one is written.
func ViolatorFor(check sdk.Kind) (sdk.Kind, bool) {
	v, known := violators[check]
	return v.violation, known
}

// Unproven is one check no stand-in can be composed to violate.
type Unproven struct {
	// Func is the check's exported identifier, and Reason says what a violator
	// would have needed.
	Func, Reason string
}

// Violation is one check and the stand-in that breaks it.
type Violation struct {
	sdk.BaseEmit
	Subject

	// Witnesses are the concrete types this guard instantiates the harness's
	// generic entry points at — the falsification's own, carried per case
	// because each case renders through its own template.
	Witnesses []sdk.Ref

	// KindName is the emit kind, and therefore the template that renders this
	// guard.
	KindName sdk.Kind

	// TestName is the guard's own identifier — `Test<Check>CanFail`.
	//
	// Suffixed rather than prefixed so `go test -run CanFail` selects the whole
	// family, which is what makes the proof runnable as a stage of its own.
	TestName string

	// Check is the assertion under proof, Method its signature, and Partner the
	// second method a spanning violation also overrides.
	Check   *Check
	Method  Method
	Partner *Method

	// Option names the stub option the violator is configured through —
	// `With<Iface><Method>`, and PartnerOption the same for a second method a
	// violation spans.
	//
	// A violation across two calls needs both: a store that ignores partitions
	// files through one and answers through the other, and overriding only the
	// write leaves the read still isolating.
	Option, PartnerOption string

	// CtorName constructs the stand-in, and FixtureCtor the derived inputs.
	CtorName, FixtureCtor string

	// Reason is what the guard reports when the check fails to reject, and
	// Because is the substring it holds the rejection to.
	//
	// The substring is what makes the proof specific. A stand-in that panicked
	// before the check's own assertion ran would satisfy a guard asserting only
	// that something failed — which is the vacuity this whole output exists to
	// catch, one level up.
	//
	// Because arrives expanded: the table's phrase carries `$M` and `$P` and
	// [expandBecause] has already put this method's and this partner's names in
	// them, so the string names one assertion line rather than a family of
	// them. See the [violators] docblock for why that distinction has teeth.
	Reason, Because string

	// HarnessPkg is where the check, the fixture and the stub live. See
	// [Falsification.HarnessPkg].
	HarnessPkg string

	// Plausible is a non-zero value per slot the check holds to its zero, for a
	// violator that has to return one.
	//
	// Derived the way the fixture's inputs are, because the claim is that a
	// subject returning something believable is caught: a stand-in returning
	// the zero satisfies the check it is meant to break.
	Plausible []FixtureValue

	// StreamElem is what a partner's channel carries, for a violator that has
	// to make one.
	//
	// Resolved here because a template cannot: the projection keeps the
	// channel's element in the ref's type arguments, and the backend's function
	// set has nothing to reach into one with.
	StreamElem sdk.Ref
}

// Kind returns [Violation.KindName].
func (v *Violation) Kind() sdk.Kind { return v.KindName }

// NeedsFixture reports whether the guard hands the check any derived value.
//
// A method taking nothing after its context is handed nothing, and Go refuses a
// declared-and-unused local — so the fixture is fetched only where it is read.
// Every parameterless lifecycle method in the corpus is this case, which is why
// it is a question rather than an assumption.
func (v *Violation) NeedsFixture() bool {
	return len(v.Check.Args) > 0 || len(v.Check.Extra) > 0
}

// plausibleReturns derives a believable value for every slot the check holds to
// its zero.
//
// The same derivation the fixture uses for parameters, applied to results —
// which is the piece the earlier version of this table said was missing. A slot
// admitting no literal makes the whole set undeliverable, because a violator
// returning the zero for one of them is a violator the check accepts.
func plausibleReturns(ctx *sdk.GeneratorContext, ck *Check) ([]FixtureValue, bool) {
	slots := ck.Method.MissReturns()
	if len(slots) == 0 {
		return nil, false
	}
	out := make([]FixtureValue, 0, len(slots))
	for _, r := range slots {
		sample, _ := golang.SampleRefFor(r.Source, r.Local, ctx.Reader)
		if !sample.OK() {
			return nil, false
		}
		out = append(out, FixtureValue{Type: r.Type, Value: sample})
	}
	return out, true
}

// streamElemOf returns what a partner's channel carries, or nil.
//
// Nil for every check whose partner hands back something else, which is every
// one but `outbox` — so a template asking for it is a template that knows it
// has a stream.
func streamElemOf(ck *Check) sdk.Ref {
	if ck.Partner == nil {
		return nil
	}
	values := ck.Partner.ValueReturns()
	if len(values) != 1 {
		return nil
	}
	elem := golang.ChanElem(values[0].Source)
	if elem == nil {
		return nil
	}
	return golang.FromNode(elem)
}

// violators maps a check kind to the stand-in that breaks it, and to the
// phrase the guard holds the rejection to.
//
// A table rather than a switch, because the thing worth reading is the pairing:
// which check each violator answers, and what the rejection has to say. The
// substring is quoted from the check's own template, so a check whose message
// changes fails its guard rather than passing it silently — which is the drift
// a second statement of the same claim invites.
//
// # Why the phrases carry $M and $P
//
// The phrase has to identify one assertion line, not a family of them. A guard
// asserting only "must accept" is satisfied by any rejection saying so — its
// own check's other assertions, a different kind's assertion on the same
// subject, the same kind's assertion about a different method. Each of those is
// a guard that passes while proving nothing, which is the vacuity this whole
// output exists to catch, arriving in the catcher.
//
// So every phrase names the method it is about, spells the partner where the
// check has one, and is quoted whole rather than truncated. The tokens expand
// through [expandBecause] at exactly the identifiers the check's own template
// interpolates, and [TestBecauseIsQuotedFromItsTemplate] holds each phrase to
// appearing verbatim in that template — which turns the paragraph above from a
// convention into something that fails a build.
var violators = map[sdk.Kind]struct {
	violation       sdk.Kind
	reason, because string

	// needsPlausible reports that the violator returns a believable value
	// rather than a zero, so the guard is only generated where one derives.
	needsPlausible bool

	// spans reports that the violation needs the check's partner overridden
	// too, so a stand-in with the write rewritten and the read left isolating
	// is not the subject the guard is about.
	spans bool
}{
	KindSmoke: {
		violation: KindViolateSmoke,
		reason:    "a method that panics on a derived value",
		because:   "$M panicked on a derived value",
	},
	KindCancel: {
		violation: KindViolateCancel,
		reason:    "a method that reports nothing for a cancelled context",
		because:   "$M must report a cancelled context as context.Canceled",
	},
	KindDeadline: {
		violation: KindViolateDeadline,
		reason:    "a method that reports nothing for an expired deadline",
		because:   "$M must report an expired deadline as context.DeadlineExceeded",
	},
	KindNilContext: {
		violation: KindViolateNilContext,
		reason:    "a method that panics on a nil context",
		because:   "$M panicked on a nil context",
	},
	KindZeroOnError: {
		violation:      KindViolateZeroOnError,
		reason:         "a method answering a believable value beside its error",
		because:        "$M must return the zero value alongside an error",
		needsPlausible: true,
	},
	KindIfAbsent: {
		violation: KindViolateIfAbsent,
		reason:    "a store that accepts every write",
		because:   "a second $M for a key already present must be refused",
	},
	KindSample: {
		violation: KindViolateSample,
		reason:    "a method that refuses what its own builder produced",
		because:   "$M must accept what $P produced",
	},
	KindValidates: {
		violation: KindViolateValidates,
		reason:    "a writer that refuses what its validator admits",
		because:   "$M must accept what $P admits",
		spans:     true,
	},
	KindWrappedVia: {
		violation: KindViolateWrappedVia,
		reason:    "a failure carrying a cause of its own",
		because:   "$M must report $P's cause where errors.Is can find it",
		spans:     true,
	},
	KindBatchSize: {
		violation: KindViolateBatchSize,
		reason:    "a reader answering once for every batch",
		because:   "$M must answer once per key it was given",
	},
	KindOutbox: {
		violation: KindViolateOutbox,
		reason:    "an outbox that accepts every record and delivers none",
		because:   "that $M appended never arrived at $P",
		spans:     true,
	},
	KindIfMatch: {
		violation: KindViolateIfMatch,
		reason:    "a writer that refuses what its own predicate admits",
		because:   "$M must accept the write $P matched",
		spans:     true,
	},
	KindMissZero: {
		violation:      KindViolateMissZero,
		reason:         "a reader answering with a plausible value for a key it does not hold",
		because:        "$M must return the zero value for an input it does not hold",
		needsPlausible: true,
	},
	KindMissFlag: {
		violation:      KindViolateMissFlag,
		reason:         "a reader answering with a plausible value beside a false flag",
		because:        "$M must return the zero value when it reports a miss",
		needsPlausible: true,
	},
	KindTimeout: {
		violation: KindViolateTimeout,
		reason:    "a method that spends twice its declared budget",
		because:   "$M ran over its declared budget",
	},
	KindCleanTeardown: {
		violation: KindViolateCleanTeardown,
		reason:    "a teardown that refuses from the first call",
		because:   "$M on a fresh subject must succeed before anything about a second call can be stated",
	},
	KindCloseIdempotent: {
		violation: KindViolateCloseIdempotent,
		reason:    "a teardown that refuses its second call",
		because:   "a second $M after a clean one must be a no-op",
	},
	KindUseAfterClose: {
		violation: KindViolateUseAfterClose,
		reason:    "an operation that answers as though nothing were closed",
		because:   "$M after $P must report the declared sentinel",
	},
	KindConcurrentSmoke: {
		violation: KindViolateConcurrentSmoke,
		reason:    "a method that panics the moment callers overlap",
		because:   "$M panicked under concurrent callers",
	},
	KindNilSafe: {
		violation: KindViolateNilSafe,
		reason:    "a method that panics on zero inputs",
		because:   "$M panicked on a zero-value input",
	},
	KindOrderAfter: {
		violation: KindViolateOrderAfter,
		reason:    "a method that accepts the call without its prerequisite",
		because:   "$M must be refused until",
	},
	KindSideEffect: {
		violation: KindViolateSideEffect,
		reason:    "a method whose effect nothing can observe",
		because:   "$M must change what $P observes",
		spans:     true,
	},
	KindHooks: {
		violation: KindViolateHooks,
		reason:    "a subject that takes a registration and forgets it",
		because:   "$M must invoke what $P registered",
		spans:     true,
	},
	KindPartition: {
		violation: KindViolatePartition,
		reason:    "a store with a single flat namespace",
		because:   "$P must return what $M wrote to its own partition, not the other's",
		spans:     true,
	},
	KindAnswerRoundTrip: {
		violation: KindViolateAnswerRoundTrip,
		reason:    "a write answering a state it did not store",
		because:   "$P must answer the state $M answered",
		spans:     true,
	},
}

// The tokens a `because` phrase carries where its check's template interpolates
// an identifier: becauseMethod for the method under check, becausePartner for
// the second method a spanning check reads through.
//
// Two characters rather than `%s`, because the phrase is read next to the
// template it was quoted from and positional verbs make that comparison a
// counting exercise. They are also what lets a phrase name the partner first —
// two checks do, and their templates report the reader rather than the writer.
const (
	becauseMethod  = "$M"
	becausePartner = "$P"
)

// expandBecause substitutes a phrase's identifiers, giving the exact sentence
// the check reports for this method.
//
// partner is empty for a check that spans nothing, and a phrase naming
// [becausePartner] under an empty partner would leave the token in the
// generated string — where the guard's own assertion fails on it, loudly,
// rather than matching something it should not. That combination is
// structurally unreachable: the kinds whose phrases name a partner are exactly
// the kinds the suite generator only selects where one resolved, which
// [TestBecauseNamesAPartnerOnlyWhereOneExists] holds it to.
func expandBecause(phrase, method, partner string) string {
	return strings.NewReplacer(becauseMethod, method, becausePartner, partner).Replace(phrase)
}

// falsificationOf builds the companion output for one interface, or reports
// that there is nothing to prove.
//
// Nothing is queued without a double: a violator is a stand-in behaving badly,
// and an interface whose source declared no `//testkit:stub` has none to
// configure. Nothing is queued without a case either — a file of nothing but
// "unproven" lines is a file that says the generator ran, which the primary
// output already says.
func falsificationOf(
	ctx *sdk.GeneratorContext, c *sdk.Provenance, iface *sdk.Interface, contract *Contract,
) (*Falsification, bool) {
	if contract.Double == nil || contract.Unfalsifiable != "" {
		return nil, false
	}

	f := &Falsification{
		BaseEmit:    sdk.EmitBaseTagged(sdk.EmitBase(c, iface), GoTestOutputTag),
		Subject:     subjectOf(iface),
		Double:      contract.Double,
		Witnesses:   contract.Double.Witnesses,
		FixtureCtor: contract.Fixture.CtorName,
		// Provisional, and deliberately wrong rather than empty: Layout has not
		// resolved the harness's package yet, and a bare name would bind to
		// whatever else is in scope instead of failing where a reader can see
		// it.
		HarnessPkg: iface.Package,
	}

	// The witness substitution, once per companion: a guard's stand-in spells
	// its overriding closure at the method's own parameter and return types,
	// and inside a Test function a type parameter is not a name.
	by := golang.WitnessBindings(iface.TypeParams, contract.Double.Witnesses)

	for _, m := range contract.Methods {
		m = witnessedMethod(m, by)
		for _, ck := range m.Checks {
			// No guard on an unknown kind. Every check kind has a violator, and
			// [TestEveryCheckKindHasAViolator] is what keeps that true — a
			// guard here would be dead code standing in for a test.
			v := violators[ck.KindName]
			plausible, derivable := plausibleReturns(ctx, ck)
			if v.needsPlausible && !derivable {
				// A stand-in returning something believable needs a literal per
				// slot, and a type admitting none leaves the guard with only
				// the zero — which is what the check already accepts. So the
				// guard would pass while proving nothing.
				f.Unproven = append(f.Unproven, Unproven{
					Func: ck.Func,
					Reason: "the stand-in has to answer with a believable value, and no " +
						"literal can be written for what this method returns",
				})
				continue
			}
			// No guard on a spanning kind arriving without its partner: a check
			// that spans two methods is selected only where the second one
			// resolved, so the nil cannot reach here.
			partnerOption, partnerName := "", ""
			if ck.Partner != nil {
				partnerOption = "With" + iface.Name + ck.Partner.Name
				partnerName = ck.Partner.Name
			}
			f.Cases = append(f.Cases, &Violation{
				BaseEmit:      sdk.EmitBaseTagged(sdk.EmitBase(c, iface), GoTestOutputTag),
				Subject:       subjectOf(iface),
				Witnesses:     contract.Double.Witnesses,
				KindName:      v.violation,
				TestName:      "Test" + ck.Func + "CanFail",
				Check:         ck,
				Method:        m,
				Option:        "With" + iface.Name + m.Name,
				PartnerOption: partnerOption,
				Partner:       witnessedPartner(ck.Partner, by),
				CtorName:      contract.Double.CtorName,
				FixtureCtor:   contract.Fixture.CtorName,
				Plausible:     plausible,
				StreamElem:    streamElemOf(ck),
				Reason:        v.reason,
				Because:       expandBecause(v.because, m.Name, partnerName),
				HarnessPkg:    iface.Package,
			})
		}
	}
	// No emptiness guard. Generate refuses an interface with no method, and
	// every method keeps at least a smoke check — which has a violator, since
	// a stand-in that panics is one any stub can be configured into. So a
	// queued harness always has something to prove.
	return f, true
}
