// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package projection

import "go.thesmos.sh/testkit/engine/suite"

// BodyKind names a body variant; the value is the variant's template
// name, composed from the one dispatch prefix.
type BodyKind string

// BodyKindPrefix namespaces body templates in the dispatch table.
const BodyKindPrefix = "body."

// The body kinds, one per variant below.
const (
	KindSmokeSurvives          BodyKind = BodyKindPrefix + "smoke-survives"
	KindCancelCall             BodyKind = BodyKindPrefix + "cancel-call"
	KindDeadlineCall           BodyKind = BodyKindPrefix + "deadline-call"
	KindNilCtxCall             BodyKind = BodyKindPrefix + "nilctx-call"
	KindZeroOnError            BodyKind = BodyKindPrefix + "zero-on-error"
	KindMixinProbe             BodyKind = BodyKindPrefix + "mixin-probe"
	KindLawLeg                 BodyKind = BodyKindPrefix + "law-leg"
	KindDifferentialLeg        BodyKind = BodyKindPrefix + "differential-leg"
	KindSimLeg                 BodyKind = BodyKindPrefix + "sim-leg"
	KindRowSugar               BodyKind = BodyKindPrefix + "row-sugar"
	KindProducedSecondarySmoke BodyKind = BodyKindPrefix + "produced-secondary-smoke"
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

// ZeroOnError asserts the non-error results are zero when the error is
// non-nil.
type ZeroOnError struct{ Call CallPlan }

// MixinProbe is a deterministic mixin claim probed through one or two
// calls (idempotent, reader-miss, and kin).
type MixinProbe struct {
	Mixin string
	Calls []CallPlan
}

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

// ProducedSecondarySmoke asserts an opened secondary closes cleanly —
// the produced type's only signature-tier coverage, per the ownership
// rules.
type ProducedSecondarySmoke struct {
	Open  CallPlan
	Close string
}

// BodyKind names the template that renders the plain survives-smoke.
func (SmokeSurvives) BodyKind() BodyKind { return KindSmokeSurvives }

// BodyKind names the template that renders the cancelled-context call.
func (CancelCall) BodyKind() BodyKind { return KindCancelCall }

// BodyKind names the template that renders the expired-deadline call.
func (DeadlineCall) BodyKind() BodyKind { return KindDeadlineCall }

// BodyKind names the template that renders the nil-context call.
func (NilCtxCall) BodyKind() BodyKind { return KindNilCtxCall }

// BodyKind names the template that renders the zero-on-error check.
func (ZeroOnError) BodyKind() BodyKind { return KindZeroOnError }

// BodyKind names the template that renders the mixin probe.
func (MixinProbe) BodyKind() BodyKind { return KindMixinProbe }

// BodyKind names the template that renders the law leg.
func (LawLeg) BodyKind() BodyKind { return KindLawLeg }

// BodyKind names the template that renders the differential leg.
func (DifferentialLeg) BodyKind() BodyKind { return KindDifferentialLeg }

// BodyKind names the template that renders the sim leg.
func (SimLeg) BodyKind() BodyKind { return KindSimLeg }

// BodyKind names the template that renders the row-sugar body.
func (RowSugar) BodyKind() BodyKind { return KindRowSugar }

// BodyKind names the template that renders the produced-secondary smoke.
func (ProducedSecondarySmoke) BodyKind() BodyKind { return KindProducedSecondarySmoke }

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
		ZeroOnError{}.BodyKind(),
		MixinProbe{}.BodyKind(),
		LawLeg{}.BodyKind(),
		DifferentialLeg{}.BodyKind(),
		SimLeg{}.BodyKind(),
		RowSugar{}.BodyKind(),
		ProducedSecondarySmoke{}.BodyKind(),
	}
}
