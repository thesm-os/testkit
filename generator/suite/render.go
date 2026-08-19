// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package suite

import (
	"strings"
	"text/template"

	"go.thesmos.sh/testkit/generator/suite/projection"
)

// renderFuncs is the rewrite's template function map, contributed to
// the backend's merged map through the plugin builder's Funcs seam —
// which is what lets the body templates live in templates/golang
// beside the plugin's structural ones: the backend parses every .tmpl
// in the plugin's FS with one merged map, so the functions must be in
// it. Registered bare; the sdk prefixes them under the plugin's name,
// and the templates call the prefixed form (suite_callExpr).
func renderFuncs() template.FuncMap {
	return template.FuncMap{
		"callExpr":       callExpr,
		"methodConst":    projection.MethodConst,
		"qualifierConst": projection.QualifierConst,
		"indexVar":       projection.IndexVar,
		"indexType":      projection.IndexType,
		"groupType":      projection.GroupType,
		"ctxArm":         ctxArmOf,
		"borrowedIdent":  func() string { return string(projection.ExprBorrowed) },
		"producedIdent":  func() string { return string(projection.ExprProduced) },
	}
}

// bodyView is one check body's rendering context: the concrete body
// variant plus the facts the emitted text needs that no projection
// carries — the subject receiver's ident, the check-name constant
// the failure attributes to, and the two return shapes only the
// method's signature knows. The emitter builds one per check as it
// walks the inventory beside the methods.
type bodyView struct {
	Recv  string
	Check string

	// Discard drops a call's results where the body only asks whether
	// the call returned: "_ =" for one result, "_, _ =" for two.
	Discard string

	// ErrBind binds the error a context body inspects — "err :=" where
	// the error is the only result, "_, err :=" where a value precedes
	// it. Uniform across both so the arms below read the same; the
	// alternative spells `return call` for one shape and a two-line
	// bind for the other, and a reader comparing two generated checks
	// then has to work out whether the difference means anything.
	ErrBind string

	Body projection.Body
}

// callExpr spells one invocation on the subject receiver, the args
// already rendered by the derivation.
func callExpr(recv string, c projection.CallPlan) string {
	args := make([]string, len(c.Args))
	for i, a := range c.Args {
		args[i] = string(a)
	}
	return recv + "." + c.Method + "(" + strings.Join(args, ", ") + ")"
}

// ctxArm is one context-family body's rendering context: the view plus
// the engine primitive that variant delegates to.
//
// The three arms differ in one identifier and agree in everything else,
// so they share a fragment rather than spelling the shape three times —
// a change to how a context body reads is then one edit rather than
// three that can disagree. The primitive is named here rather than
// carried on the projection because it is a fact about the emitted
// spelling, not about the check.
type ctxArm struct {
	View      bodyView
	Primitive string

	// Vocab is the package the primitive is declared in, carried so the
	// fragment can register the import through the canonical helper.
	Vocab string
}

// ctxArmOf pairs a body view with its primitive.
func ctxArmOf(v bodyView, primitive string) ctxArm {
	return ctxArm{View: v, Primitive: primitive, Vocab: Vocab}
}
