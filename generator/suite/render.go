// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package suite

import (
	"strings"
	"text/template"

	"go.thesmos.sh/eidos/sdk"

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
		"callExpr":          callExpr,
		"methodConst":       projection.MethodConst,
		"qualifierConst":    projection.QualifierConst,
		"harnessName":       projection.HarnessName,
		"subjectType":       subjectType,
		"withParam":         withParam,
		"checksName":        projection.ChecksName,
		"configName":        projection.ConfigName,
		"defaultConfigName": projection.DefaultConfigName,
		"fixtureIdent":      func() string { return string(projection.ExprFixture) },
		"runName":           projection.RunName,
		"proveName":         projection.ProveName,
		"runOptName":        projection.RunOptName,
		"runConfigName":     projection.RunConfigName,
		"dropOptName":       projection.DropOptName,
		"withoutName":       projection.WithoutName,
		"veneerName":        projection.VeneerTypeName,
		"indexPathName":     projection.IndexPathName,
		"dropHintName":      projection.DropHintName,
		"indexVar":          projection.IndexVar,
		"indexType":         projection.IndexType,
		"groupType":         projection.GroupType,
		"ctxArm":            ctxArmOf,
		"borrowedIdent":     func() string { return string(projection.ExprBorrowed) },
		"producedIdent":     func() string { return string(projection.ExprProduced) },
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

	// ErrBind binds the error a context body inspects — "_, err :="
	// where a value precedes the error — and is EMPTY where the error
	// is the only result, which the packs return directly.
	//
	// Empty rather than "err :=" because a bind whose only use is the
	// next line is a local the reader has to follow: `return s.Put(ctx,
	// fx.Put())` says the whole thing. The two arms are the validated
	// spelling, and the difference between them is exactly the
	// difference between the signatures.
	ErrBind string

	// Draws says the emitted assert function takes the run's fixture,
	// which it does wherever the method has a drawable argument — the
	// packs' own rule, `storeAssertLenSmoke(tb, s)` beside
	// `storeAssertGetSmoke(tb, s, fx)`.
	//
	// Read from the method rather than from the body's arguments, and
	// so an over-approximation: the borrow arm substitutes the borrowed
	// local for a drawn one and may take the fixture without reading
	// it. That direction is free — an unused parameter compiles — while
	// the other is a body naming a local nothing declared.
	Draws bool

	// Method is the name the failure messages spell. The check constant
	// beside it is what the engine primitives take; a message is prose
	// and says the method the way the source declares it.
	Method string

	// ValueBind binds a call's results where the body judges the first
	// of them — "got, err :=", widening by one blank per extra result.
	ValueBind string

	// Pool is the config field the miss body's skip tells a consumer to
	// seed, empty on every other body.
	Pool string

	// NeedsCtx says the body's calls take a context, so it declares one;
	// HasErr says the method reports failure at all, which decides
	// whether a body can judge an error or only a value.
	NeedsCtx bool
	HasErr   bool

	// ValueDiscard blanks the results after the first, for a body that
	// judges one value from a method reporting no error.
	ValueDiscard string

	// ErrStmt binds the error inside an if-statement's init — `err :=`,
	// or `_, err :=` past each value. Never empty where the method
	// reports one, because a body judging the error in the condition
	// has nowhere else to bind it.
	ErrStmt string

	// Sentinel is the error a declared miss reports, resolved to a
	// reference so the body naming it registers the import. Nil where
	// the declaration names none, which is a different body.
	Sentinel *sdk.Expr

	// Zero says how this result's zero is compared, and ZeroType is the
	// type a declared zero is declared of — a rendered reference for a
	// named type, the bare word for a predeclared one. Both are read
	// from the classification the claim is worded from.
	Zero     ZeroShape
	ZeroType *sdk.Expr
	ZeroWord string

	Body projection.Body
}

// ZeroIsNil lets the template branch on the classification without
// knowing its numbering.
func (v bodyView) ZeroIsNil() bool { return v.Zero == ZeroNil }

// callExpr spells one invocation on the subject receiver, the args
// already rendered by the derivation.
func callExpr(recv string, c projection.CallPlan) string {
	args := make([]string, len(c.Args))
	for i, a := range c.Args {
		args[i] = string(a)
	}
	return recv + "." + c.Method + "(" + strings.Join(args, ", ") + ")"
}

// ctxArm is one context-family body's rendering context: the call, how
// its error is bound, and the engine primitive that variant delegates
// to.
//
// The three arms differ in one identifier and agree in everything else,
// so they share a fragment rather than spelling the shape three times.
// It takes the fields it needs rather than a view type, because the
// same fragment renders from the emit node in a run and from the
// parse-only harness in a test, and a fragment bound to one of those
// shapes cannot serve the other.
type ctxArm struct {
	Recv, Check, ErrBind string
	Call                 projection.CallPlan
	Primitive            string

	// Vocab is the package the primitive is declared in, carried so the
	// fragment registers the import through the canonical helper.
	Vocab string
}

// ctxArmOf pairs one call with the primitive that judges it.
func ctxArmOf(recv, check, errBind string, call projection.CallPlan, primitive string) ctxArm {
	return ctxArm{
		Recv: recv, Check: check, ErrBind: errBind,
		Call: call, Primitive: primitive, Vocab: Vocab,
	}
}

// subjectType spells the interface as the generated declarations name
// it: the alias for a concrete interface, the qualified reference plus
// its arguments for a generic one.
//
// A generic interface has no witnessed instantiation to alias — `type
// Store = generic.Store` names a type that does not exist without its
// arguments — so the file spells what the alias was shorthand for. The
// packs never reach this: every validated interface is concrete.
func subjectType(c *Contract) string {
	if len(c.TypeParams) == 0 {
		return c.IfaceName
	}
	return c.IfaceName + c.TypeArgs
}

// withParam appends one parameter to a rendered type-parameter list.
//
// The harness declares the subject's own parameters ahead of T, because
// T is constrained by the interface and Go admits no forward reference:
// `StoreHarness[K comparable, V any, T Store[K, V]]`. The list itself
// is the backend's to render — only it knows how to spell a
// constraint — so this takes what it rendered and adds to it, which
// keeps the composition here and the spelling there.
func withParam(rendered, extra string) string {
	if rendered == "" {
		return "[" + extra + "]"
	}
	return strings.TrimSuffix(rendered, "]") + ", " + extra + "]"
}
