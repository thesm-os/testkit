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

		call := callOf(f, m)
		plans = append(plans, smokePlan(f, m, call, seeded))

		if !f.CtxDeclared || !m.TakesContext() {
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
				Body:        projection.ZeroOnError{Call: call},
				Falsifiable: vocab.Proven(),
				Defect:      projection.EchoBesideError{Option: projection.OptionName(f.Name, m.Name)},
			})
		}
	}
	return plans, refusals
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
