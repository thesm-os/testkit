// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package projection is the suite plugin's data model: the blueprint a
// generation run computes and the templates render. Everything here is
// build-time — a CheckPlan lives for milliseconds inside `testkit run`
// and is never seen by a consumer; the generated file it describes
// constructs the runtime's own [suite.Check] values instead.
//
// One inventory sources every artifact. Claim text, probe sets, lock
// rows, the typed index, the proofs table and the selfcheck census are
// all projections of the same nodes, which is what makes "a claim is
// exactly as wide as its assertion" a property of the data flow rather
// than a rule enforced by review.
package projection

import (
	"errors"
	"fmt"

	"go.thesmos.sh/testkit/engine/suite"
)

// IDPlan spells one check identity in the grammar's terms rather than
// as a string, so the emitter cannot mint an ID shape the runtime
// would refuse. Exactly one of Method or Family is set: method-scoped
// IDs render "Method/seg", family-scoped IDs render
// "family/qualifier/seg" — and the qualifier is unconditional for
// family scopes, per the uniform-qualification ruling.
type IDPlan struct {
	Method    string // "Append" -> "Append/<seg>"
	Family    string // suite.FamilyModel etc. -> "<family>/<qualifier>/<seg>"
	Qualifier string // the interface token; required with Family
	Seg       string
}

// Render produces the runtime ID through the engine vocabulary — the
// one home of the grammar.
//
// A malformed plan is a deriver bug rather than consumer input, and it
// is still reported rather than panicked: [Inventory.Verify] is the
// seam that holds a run to its own invariants, and a panic here would
// jump straight over it — taking every other interface's output down
// for one deriver's mistake instead of failing the interface being
// derived, with the plan named.
func (p IDPlan) Render() (suite.ID, error) {
	switch {
	case p.Method != "" && p.Family != "":
		return "", fmt.Errorf("projection: ID plan sets both Method %q and Family %q", p.Method, p.Family)
	case p.Method != "":
		return suite.MethodID(p.Method, p.Seg), nil
	case p.Family != "":
		if p.Qualifier == "" {
			return "", fmt.Errorf(
				"projection: family ID %s/%s lacks its interface qualifier; qualification is unconditional",
				p.Family,
				p.Seg,
			)
		}
		return suite.FamilyID(p.Family, p.Qualifier, p.Seg), nil
	default:
		return "", errors.New("projection: empty ID plan")
	}
}

// CheckPlan is the blueprint for one emitted check. The closed Body
// and Defect variant sets are deliberate: a family that wants "one
// more optional field" adds a variant and a template instead, visibly,
// which is the guard against this node rotting into a god-struct.
type CheckPlan struct {
	ID    IDPlan
	Class suite.Class
	Claim string

	// Needs names the capability doors the runtime gate consults;
	// values are rendered into the check's Caps literal.
	Needs []NeedPlan

	// Body is how the check asserts — exactly one variant.
	Body Body

	// Falsifiable carries the claim about the claim: Proven demands a
	// Defect; Argued demands a reason and forbids one. The inventory's
	// census refuses any other combination.
	Falsifiable suite.Falsifiability

	// Defect is the planted implementation that must red this check —
	// nil exactly when the check is Argued.
	Defect Defect

	// Binds names the assertion bodies the check delegates to, and
	// renders into the lock's fourth column, which is what makes
	// narrowing a probe set diff.
	Binds []Bind
}

// NeedPlan is one capability door: the runtime capability constant
// plus the rendered value literal, empty for presence-only doors.
type NeedPlan struct {
	Capability suite.Capability
	Value      Expr
}

// Body is the closed set of check-body shapes. One template per kind;
// the kind IS the template's name, and the census holds the two
// registries equal.
type Body interface{ BodyKind() BodyKind }

// Defect is the closed set of planted-defect shapes, mirroring the
// proofs rules in derivation-rules.md.
type Defect interface{ DefectKind() DefectKind }
