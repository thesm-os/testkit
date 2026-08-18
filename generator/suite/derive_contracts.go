// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package suite

import (
	"go.thesmos.sh/eidos/node"

	vocab "go.thesmos.sh/testkit/engine/suite"
	"go.thesmos.sh/testkit/generator/suite/projection"
)

// The contract arm of derivation: rules keyed on the contract stamps
// the annotator resolved, adjusting the signature families where a
// contract changes what a method's own claim must say. The direct
// contract checks (if-absent, if-match, outbox) and the borrowed-input
// smoke stay pending — the census and the design doc's frontier name
// them.

// openerSmoke overrides the smoke for a producing method: the cursor
// contract's open role answers a handle the smoke must close, because
// the opener owns what it opens and a leaked handle in the suite's own
// smoke would be the harness teaching the leak. The produced type
// itself carries no suite directive, so its absence of smokes needs no
// rule.
//
// A stamped opener without a resolved close partner falls back to the
// plain smoke rather than refusing: the contract schema owns partner
// completeness, and eidos reports that gap at annotation time where
// the author can act on it.
func openerSmoke(f Iface, m Method, call projection.CallPlan) (projection.CheckPlan, bool) {
	if !m.HasContractRole(ContractCursor, ContractCursorOpen) {
		return projection.CheckPlan{}, false
	}
	closeName := m.ContractPartner(ContractCursor, ContractCursorClose)
	if closeName == "" {
		return projection.CheckPlan{}, false
	}
	return projection.CheckPlan{
		ID:          projection.IDPlan{Method: m.Name, Seg: vocab.SegSmoke},
		Class:       vocab.ClassSmoke,
		Claim:       OpenerSmokeClaim(m, ContractCursor),
		Body:        projection.SmokeSurvives{Call: call, CloseProduced: closeName},
		Falsifiable: vocab.Proven(),
		Defect:      projection.StubPanic{Option: projection.OptionName(f.Name, m.Name)},
	}, true
}

// borrowSmoke overrides the smoke for the pool contract's put role:
// its input is pool-produced, nothing the fixture can derive, so the
// smoke borrows from the get sibling first and returns what it
// borrowed. Answers before the undeliverable-args refusal, because
// the produced draw is the borrow's to supply. Without a get sibling
// or a parameter taking the produced type there is nothing to borrow,
// and the ordinary refusal names the gap instead.
func borrowSmoke(f Iface, m Method) (projection.CheckPlan, bool) {
	if !m.HasContractRole(ContractPool, ContractPoolPut) {
		return projection.CheckPlan{}, false
	}
	producer := roleMethod(f.Methods, ContractPool, ContractPoolGet)
	if producer == nil {
		return projection.CheckPlan{}, false
	}
	call, matched := borrowedCall(f, m, producedType(*producer))
	if !matched {
		return projection.CheckPlan{}, false
	}
	return projection.CheckPlan{
		ID:          projection.IDPlan{Method: m.Name, Seg: vocab.SegSmoke},
		Class:       vocab.ClassSmoke,
		Claim:       BorrowSmokeClaim(m),
		Body:        projection.SmokeSurvives{Call: call, Borrow: callOf(f, *producer)},
		Falsifiable: vocab.Proven(),
		Defect:      projection.StubPanic{Option: projection.OptionName(f.Name, m.Name)},
	}, true
}

// roleMethod finds the sibling filling a contract role, nil when none
// does.
func roleMethod(methods []Method, contract, role string) *Method {
	for i := range methods {
		if methods[i].HasContractRole(contract, role) {
			return &methods[i]
		}
	}
	return nil
}

// producedType is the producer's non-error answer — what the borrow
// binds and the returning call passes back. Nil when the producer
// answers nothing, which no valid pool schema stamps.
func producedType(producer Method) *node.TypeRef {
	values := producer.ValueReturns()
	if len(values) == 0 {
		return nil
	}
	return values[0].Source
}

// borrowedCall renders the returning method's invocation: the context
// first, the borrowed local where a parameter takes the produced
// type, the fixture draw otherwise. False when no parameter takes it.
func borrowedCall(f Iface, m Method, produced *node.TypeRef) (projection.CallPlan, bool) {
	var args []projection.Expr
	if m.TakesContext() {
		args = append(args, projection.ExprCtx)
	}
	matched := false
	for i, p := range m.CallArgs() {
		if sameNamed(p.Source, produced) {
			args = append(args, projection.ExprBorrowed)
			matched = true
			continue
		}
		if i < len(m.ArgFields) {
			args = append(args, projection.FixtureCall(f.Token, m.ArgFields[i]))
		}
	}
	return projection.CallPlan{Method: m.Name, Args: args}, matched
}

// sameNamed reports whether two refs name one declared type. Named
// refs only: the borrow correspondence is by declared type, and a
// composite cannot be the produced handle.
func sameNamed(a, b *node.TypeRef) bool {
	if a == nil || b == nil {
		return false
	}
	return a.TypeKind == node.TypeRefNamed && b.TypeKind == node.TypeRefNamed &&
		a.Name == b.Name && a.Package == b.Package
}
