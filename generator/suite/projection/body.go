// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package projection

import "go.thesmos.sh/testkit/engine/suite"

// BodyKind names a body variant; the value is the variant's template
// name, composed from the one dispatch prefix.
type BodyKind string

// BodyKindPrefix namespaces body templates in the dispatch table.
//
// The plugin's own name leads it because the backend parses every
// plugin's templates into one map: a kind is a {{define}} name in a
// shared namespace, so an unprefixed "body.smoke-survives" would
// collide with any other plugin that reached for the obvious word.
// Same rule the emitted node kinds follow.
const BodyKindPrefix = "suite.body."

// The body kinds, one per variant below.
const (
	KindSmokeSurvives   BodyKind = BodyKindPrefix + "smoke-survives"
	KindCancelCall      BodyKind = BodyKindPrefix + "cancel-call"
	KindDeadlineCall    BodyKind = BodyKindPrefix + "deadline-call"
	KindNilCtxCall      BodyKind = BodyKindPrefix + "nilctx-call"
	KindZeroOnMiss      BodyKind = BodyKindPrefix + "zero-on-miss"
	KindZeroOnCancel    BodyKind = BodyKindPrefix + "zero-on-cancel"
	KindRepeatProbe     BodyKind = BodyKindPrefix + "repeat-probe"
	KindMissProbe       BodyKind = BodyKindPrefix + "miss-probe"
	KindHitProbe        BodyKind = BodyKindPrefix + "hit-probe"
	KindCountProbe      BodyKind = BodyKindPrefix + "count-probe"
	KindLawLeg          BodyKind = BodyKindPrefix + "law-leg"
	KindDifferentialLeg BodyKind = BodyKindPrefix + "differential-leg"
	KindSimLeg          BodyKind = BodyKindPrefix + "sim-leg"
	KindRowSugar        BodyKind = BodyKindPrefix + "row-sugar"
)

// CallPlan spells one method invocation a body makes: the method name
// and the rendered argument expressions, fixture draws included.
type CallPlan struct {
	Method string
	Args   []Expr
}

// The body variants. Each is exactly the data its template needs and
// nothing speculative — a variant grows a field the day a template
// renders it.

// SmokeSurvives asserts the call returns without panicking; a produced
// handle it opens is closed in the same body. One variant carries the
// three smoke shapes — plain, opener, borrower — because all three
// state one claim family and differ only in prologue and epilogue.
type SmokeSurvives struct {
	Call CallPlan
	// Borrow is the producing sibling called first when the smoked
	// method's input is pool-produced: its result feeds the smoked
	// call, and a failed borrow returns without judgment — the
	// producer's own smoke owns that path.
	Borrow CallPlan
	// CloseProduced names the produced handle's release method when
	// the smoked method answers one — the opener owns what it opens.
	CloseProduced string
}

// CancelCall asserts an already-cancelled context is reported as
// context.Canceled.
type CancelCall struct{ Call CallPlan }

// DeadlineCall asserts an expired deadline is reported as
// context.DeadlineExceeded.
type DeadlineCall struct{ Call CallPlan }

// NilCtxCall asserts a nil context returns an error rather than
// panicking or answering.
type NilCtxCall struct{ Call CallPlan }

// ZeroOnMiss asserts the non-error results are zero when a draw that
// nothing seeded produces the declared miss sentinel.
//
// Split from [ZeroOnCancel] because the two induce their error
// differently and so are different statement sequences, not one body
// with a mode: this one draws the alternate member and skips when the
// subject answers anyway, that one cancels a context first. A single
// variant covering both was one variant covering three shapes, which is
// how the closed set stops meaning anything.
type ZeroOnMiss struct {
	// Call draws the ALTERNATE member — the error this check inspects
	// only happens for an input nothing wrote.
	Call CallPlan

	// Pool is the config field a consumer seeds to make the miss a
	// miss, named in the skip so a run that proves nothing says what
	// would make it prove something.
	Pool string
}

// ZeroOnCancel asserts the non-error results are zero when a cancelled
// context produces the error.
//
// The form for a method whose inputs cannot miss — one that takes none,
// or one no sentinel declares a miss for. A cancelled context is the
// only error every context-taking method can be made to report.
type ZeroOnCancel struct{ Call CallPlan }

// RepeatProbe calls a method twice and judges the second: the first
// call is a precondition and fails the check outright, the second is
// the claim.
//
// The asymmetry is the whole shape and is why this is not a list of
// calls: a body that treated both the same would report a subject
// whose first Close failed as a subject that is not idempotent, which
// is a different fault with a different fix.
type RepeatProbe struct{ Call CallPlan }

// MissProbe reads an input nothing supplied and judges the answer.
//
// Sentinel is the error a miss is declared to report, empty where the
// declaration names none — and the two are different bodies rather
// than one with an optional field: with a sentinel the claim is
// errors.Is against it, without one the claim is that the answer is
// the zero, and the second needs the call to have succeeded first.
type MissProbe struct {
	Call     CallPlan
	Sentinel Expr
}

// HitProbe reads back what the run seeded and judges the answer
// against it.
//
// Only derivable where the interface seeds, which is what supplies
// something to read back.
type HitProbe struct{ Call CallPlan }

// CountProbe judges an aggregate against the size of what was seeded.
type CountProbe struct{ Call CallPlan }

// LawLeg delegates to legs.Law with the named engine laws; Laws also
// feeds the plan's Binds. Probes maps a law's probe name to its call,
// for multi-probe laws whose claim spans several methods.
type LawLeg struct {
	Laws   []Bind
	Probes map[string]CallPlan
	// Extra carries leg options (a history reset, a produced lift)
	// as rendered expressions.
	Extra []Expr
}

// DifferentialLeg delegates to legs.Differential over the derived
// reference and the action vocabulary.
type DifferentialLeg struct {
	// The model file owns actions and references; the suite file's
	// check only names the assert function the model file exports.
	AssertFunc string
}

// SimKind names a sim-tier leg; the values are the runtime's own
// segment constants, so the sim vocabulary keeps one home.
type SimKind string

// The sim kinds, from the engine vocabulary.
const (
	SimRecovery = SimKind(suite.SegRecovery)
	SimCrash    = SimKind(suite.SegCrash)
	SimFault    = SimKind(suite.SegFault)
)

// SimLeg is a recovery/crash/fault check body over the Recover seam.
type SimLeg struct {
	Kind SimKind
}

// RowSugar is the consumer extension seam: the typed row table and its
// bind, not a derived assertion of its own.
type RowSugar struct{}

// BodyKind names the template that renders the plain survives-smoke.
func (SmokeSurvives) BodyKind() BodyKind { return KindSmokeSurvives }

// BodyKind names the template that renders the cancelled-context call.
func (CancelCall) BodyKind() BodyKind { return KindCancelCall }

// BodyKind names the template that renders the expired-deadline call.
func (DeadlineCall) BodyKind() BodyKind { return KindDeadlineCall }

// BodyKind names the template that renders the nil-context call.
func (NilCtxCall) BodyKind() BodyKind { return KindNilCtxCall }

// BodyKind names the template that renders the miss-induced zero check.
func (ZeroOnMiss) BodyKind() BodyKind { return KindZeroOnMiss }

// BodyKind names the template that renders the cancel-induced zero check.
func (ZeroOnCancel) BodyKind() BodyKind { return KindZeroOnCancel }

// BodyKind names the template that renders the repeat probe.
func (RepeatProbe) BodyKind() BodyKind { return KindRepeatProbe }

// BodyKind names the template that renders the miss probe.
func (MissProbe) BodyKind() BodyKind { return KindMissProbe }

// BodyKind names the template that renders the seeded-hit probe.
func (HitProbe) BodyKind() BodyKind { return KindHitProbe }

// BodyKind names the template that renders the seeded-count probe.
func (CountProbe) BodyKind() BodyKind { return KindCountProbe }

// BodyKind names the template that renders the law leg.
func (LawLeg) BodyKind() BodyKind { return KindLawLeg }

// BodyKind names the template that renders the differential leg.
func (DifferentialLeg) BodyKind() BodyKind { return KindDifferentialLeg }

// BodyKind names the template that renders the sim leg.
func (SimLeg) BodyKind() BodyKind { return KindSimLeg }

// BodyKind names the template that renders the row-sugar body.
func (RowSugar) BodyKind() BodyKind { return KindRowSugar }

// BodyKinds enumerates every registered body variant. The template
// census holds this list and the embedded template set equal, so an
// unregistered variant or an orphaned template is a build failure, not
// a render error in a consumer's run.
func BodyKinds() []BodyKind {
	return []BodyKind{
		SmokeSurvives{}.BodyKind(),
		CancelCall{}.BodyKind(),
		DeadlineCall{}.BodyKind(),
		NilCtxCall{}.BodyKind(),
		ZeroOnMiss{}.BodyKind(),
		ZeroOnCancel{}.BodyKind(),
		RepeatProbe{}.BodyKind(),
		MissProbe{}.BodyKind(),
		HitProbe{}.BodyKind(),
		CountProbe{}.BodyKind(),
		LawLeg{}.BodyKind(),
		DifferentialLeg{}.BodyKind(),
		SimLeg{}.BodyKind(),
		RowSugar{}.BodyKind(),
	}
}
