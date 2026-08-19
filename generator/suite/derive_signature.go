// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package suite

import (
	vocab "go.thesmos.sh/testkit/engine/suite"
	"go.thesmos.sh/testkit/generator/suite/projection"
)

// Signature derives the per-method families. The rules, from
// derivation-rules.md:
//
//   - one smoke per method, always;
//   - the context families (cancel, nilcontext, deadline) only under
//     the //testkit:ctx directive, and only for context-taking
//     methods — context semantics are a declared claim, never a
//     signature inference;
//   - deadline never on a teardown-shaped method: an expired deadline
//     on the way out is not a claim the contract makes;
//   - zero-on-error only under the directive, for methods answering a
//     value beside their error — absent the directive the family has
//     no derivable error source;
//   - a method whose draws the fixture cannot supply refuses its
//     whole family set in one refusal, naming the WithFixture remedy.
type Signature struct{}

// Name implements [Deriver].
func (Signature) Name() DeriverName { return DeriverSignature }

// Derive implements [Deriver].
func (Signature) Derive(f Iface) ([]projection.CheckPlan, []Refusal) {
	var plans []projection.CheckPlan
	var refusals []Refusal
	seeded := f.seeded()

	for _, m := range f.Methods {
		if plan, borrowed := borrowSmoke(f, m); borrowed {
			// The produced draw is the borrow's to supply, so the
			// borrow arm answers before the undeliverable refusal.
			// Context families on a borrowed method are an open rule
			// (design-doc frontier); no corpus contract declares both.
			plans = append(plans, plan)
			continue
		}
		if r, refused := argsRefusal(DeriverSignature, f, m, "'s signature checks"); refused {
			refusals = append(refusals, r)
			continue
		}

		call := callOf(m)
		plans = append(plans, smokePlan(f, m, call, seeded))

		if !m.TakesContext() || !m.ReturnsError() {
			// The engine primitives judge a `func(ctx) error`, so a
			// method with no error channel has nothing to report a
			// cancellation through. The directive used to imply this —
			// an author claiming context semantics was claiming an
			// error to carry them — and deriving from the shape has to
			// say it outright.
			continue
		}
		plans = append(
			plans,
			ctxPlan(f, m, vocab.SegCancel, vocab.ClassCancel, CancelClaim(m), projection.CancelCall{Call: call}),
			ctxPlan(
				f,
				m,
				vocab.SegNilContext,
				vocab.ClassNilContext,
				NilCtxClaim(m),
				projection.NilCtxCall{Call: call},
			),
		)
		if !teardownShaped(m) {
			plans = append(
				plans,
				ctxPlan(
					f,
					m,
					vocab.SegDeadline,
					vocab.ClassDeadline,
					DeadlineClaim(m),
					projection.DeadlineCall{Call: call},
				),
			)
		}
		if m.ReturnsError() && len(m.ValueReturns()) > 0 && !m.HasMixin(MixinTotal) {
			plans = append(plans, projection.CheckPlan{
				ID:          projection.IDPlan{Method: m.Name, Seg: vocab.SegZeroValue},
				Class:       vocab.ClassZeroValue,
				Claim:       ZeroOnErrorClaim(m),
				Body:        zeroBody(f, m, call),
				Falsifiable: vocab.Proven(),
				Defect:      projection.EchoBesideError{Option: projection.OptionName(f.Name, m.Name)},
			})
		}
	}
	return plans, refusals
}

// zeroBody picks how the check induces the error it inspects.
//
// A draw that misses, where the declaration says a miss is an error:
// the corpus spells that with a `notfound=` sentinel, and only then
// does handing the method an unwritten input produce anything to
// inspect. Otherwise a cancelled context, which is the one error every
// context-taking method can be made to report.
//
// Not keyed on whether the method takes an input, which is the reading
// that looks right and is not: the bus's Subscribe takes a topic and
// declares no miss, so an unsubscribed topic answers normally and a
// check drawing one would skip every run.
func zeroBody(f Iface, m Method, call projection.CallPlan) projection.Body {
	if _, declared := m.MixinParam(MixinTTL, MixinTTLNotFound); declared && m.HasInput() {
		return projection.ZeroOnMiss{
			Call: missCall(f, m),
			Pool: missPool(f, m),
		}
	}
	return projection.ZeroOnCancel{Call: call}
}

// missCall is [callOf] against the alternate members: the draw nothing
// wrote, which is what makes the miss a miss.
func missCall(f Iface, m Method) projection.CallPlan {
	var args []projection.Expr
	if m.TakesContext() {
		args = append(args, projection.ExprCtx)
	}
	for _, field := range fixtureArgs(f.Fixture, m, true) {
		args = append(args, projection.FixtureCall(projection.ExprFixture, field))
	}
	return projection.CallPlan{Method: m.Name, Args: args}
}

// missPool is the config field a consumer seeds to make the drawn miss
// answer — the remedy the skip names.
func missPool(f Iface, m Method) string {
	fields := m.ArgFields
	if len(fields) == 0 {
		return ""
	}
	return projection.ConfigName(f.Name) + "." + projection.PoolFieldName(fields[0])
}

// smokePlan is the always-derived family: proven by the panicking
// double. A contract can override what the smoke must say — the
// cursor opener closes what it opens — and the contract arm answers
// first so the smoke ID stays single-sourced.
func smokePlan(f Iface, m Method, call projection.CallPlan, seeded bool) projection.CheckPlan {
	if plan, overridden := openerSmoke(f, m, call); overridden {
		return plan
	}
	return projection.CheckPlan{
		ID:          projection.IDPlan{Method: m.Name, Seg: vocab.SegSmoke},
		Class:       vocab.ClassSmoke,
		Claim:       SmokeClaim(m, seeded),
		Body:        projection.SmokeSurvives{Call: call},
		Falsifiable: vocab.Proven(),
		Defect:      projection.StubPanic{Option: projection.OptionName(f.Name, m.Name)},
	}
}

// ctxPlan is the shared shape of the context families: same call,
// family-specific body and wording, proven by the context-ignoring
// double — except nilcontext, whose claim's stronger arm (returns an
// error) is proven by the accepting double.
func ctxPlan(
	f Iface,
	m Method,
	seg string,
	class vocab.Class,
	claim string,
	body projection.Body,
) projection.CheckPlan {
	var defect projection.Defect = projection.CtxSwap{Option: projection.OptionName(f.Name, m.Name)}
	if seg == vocab.SegNilContext {
		defect = projection.AcceptsNil{Option: projection.OptionName(f.Name, m.Name)}
	}
	return projection.CheckPlan{
		ID:          projection.IDPlan{Method: m.Name, Seg: seg},
		Class:       class,
		Claim:       claim,
		Body:        body,
		Falsifiable: vocab.Proven(),
		Defect:      defect,
	}
}
