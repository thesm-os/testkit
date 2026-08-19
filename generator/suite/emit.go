// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package suite

import (
	"sort"
	"strings"

	"go.thesmos.sh/eidos/lang/golang"
	"go.thesmos.sh/eidos/node"
	"go.thesmos.sh/eidos/sdk"

	vocab "go.thesmos.sh/testkit/engine/suite"
	"go.thesmos.sh/testkit/generator/suite/projection"
)

// CheckEmit is one derived check as its template renders it.
//
// The node the backend dispatches on: its kind IS the body variant's
// template name, so an unrendered variant fails by name rather than
// emitting nothing. The plan says what the check asserts; the fields
// beside it say how this file spells it, and every one of them is a
// fact about the METHOD rather than about the check — which is why
// they are computed here, where the method is still in hand, instead
// of being carried through the projection.
type CheckEmit struct {
	sdk.BaseEmit
	bodyView

	// Plan is the derived check itself: identity, class, claim, binds.
	// The rows read it; the body templates read the view above.
	Plan projection.CheckPlan

	// The spellings the row needs, resolved once here rather than
	// composed in the template: a row naming an accessor its own index
	// does not declare is a compile error a consumer meets.
	accessor, assertName, classConst string
}

// Group is the index member this check sits under, so a row names it
// through the same tree a consumer drops it through.
func (c *CheckEmit) Group() string { return golang.ExportedName(c.Plan.ID.Method) }

// Accessor is this check's entry point within that member.
func (c *CheckEmit) Accessor() string { return c.accessor }

// AssertName is the identifier of the function carrying this check.
func (c *CheckEmit) AssertName() string { return c.assertName }

// ClassConst is the engine identifier the check's class is declared
// under, so the row names the class rather than repeating its slug.
func (c *CheckEmit) ClassConst() string { return c.classConst }

// Kind returns the plan's body variant, which is its template's name.
func (c *CheckEmit) Kind() sdk.Kind { return sdk.Kind(c.Plan.Body.BodyKind()) }

// rendered is the body variants whose templates exist today.
//
// A body with no template fails the backend's dispatch by name, which
// is the guard working — so the rows carry only what can be rendered,
// and [WithheldBodies] names the rest in the generated file rather than
// letting a reader infer coverage from silence.
func rendered() map[projection.BodyKind]bool {
	return map[projection.BodyKind]bool{
		projection.KindSmokeSurvives: true,
		projection.KindCancelCall:    true,
		projection.KindDeadlineCall:  true,
		projection.KindNilCtxCall:    true,
		projection.KindZeroOnMiss:    true,
		projection.KindZeroOnCancel:  true,
		projection.KindRepeatProbe:   true,
		projection.KindMissProbe:     true,
	}
}

// checkEmitsOf pairs every plan with the method it is about.
//
// The pairing is the seam the whole emission turns on. A plan names its
// method as a string, deliberately — the projection is unit-testable
// without the pipeline and holds no source node — so the facts a body
// needs from the signature are resolved here, once per check, against
// the method set the run already projected.
//
// A plan whose method is not in the set is dropped rather than emitted
// half-spelled: it can only come from a deriver naming something the
// interface does not declare, and a call to a method that is not there
// fails in the consumer's build rather than in this run. Family-scoped
// plans name no method and carry no body of ours, so they are not here
// at all.
func checkEmitsOf(base sdk.BaseEmit, iface Iface, inv projection.Inventory) []*CheckEmit {
	byName := make(map[string]Method, len(iface.Methods))
	for _, m := range iface.Methods {
		byName[m.Name] = m
	}

	out := make([]*CheckEmit, 0, len(inv.Checks))
	for _, plan := range inv.Checks {
		if plan.ID.Method == "" {
			continue
		}
		m, found := byName[plan.ID.Method]
		if !found {
			continue
		}
		if !rendered()[plan.Body.BodyKind()] {
			continue
		}
		acc, err := projection.AccessorOf(plan.ID)
		if err != nil {
			// The index refused to name it, so a row naming it would
			// spell a method the index does not declare. IndexOf
			// reports the same refusal by name; this drops quietly
			// rather than saying it twice.
			continue
		}
		class, named := vocab.ClassConst(plan.Class)
		if !named {
			continue
		}
		view := viewOf(iface, m)
		view.Body = plan.Body
		if miss, ok := plan.Body.(projection.ZeroOnMiss); ok {
			view.Pool = miss.Pool
		}
		if probe, ok := plan.Body.(projection.MissProbe); ok && probe.Sentinel != "" {
			view.Sentinel = sentinelRef(iface, string(probe.Sentinel))
		}
		out = append(out, &CheckEmit{
			BaseEmit:   base,
			bodyView:   view,
			Plan:       plan,
			accessor:   acc.Name,
			assertName: projection.AssertName(iface.Token, plan.ID.Method, plan.ID.Seg),
			classConst: class,
		})
	}
	return out
}

// viewOf spells the facts a body needs from one method's signature.
func viewOf(iface Iface, m Method) bodyView {
	return bodyView{
		Recv:         receiverIdent(iface),
		Check:        projection.MethodConst(iface.Token, m.Name),
		Discard:      discardOf(m),
		ErrBind:      errBindOf(m),
		Draws:        len(m.ArgFields) > 0,
		Method:       m.Name,
		ValueBind:    valueBindOf(m),
		ErrStmt:      errStmtOf(m),
		ValueDiscard: valueDiscardOf(m),
		NeedsCtx:     m.TakesContext(),
		HasErr:       m.ReturnsError(),
		Zero:         ZeroShapeOf(m),
		ZeroType:     zeroTypeOf(m),
		ZeroWord:     zeroWordOf(m),
	}
}

// receiverIdent is the local a body calls the subject through.
//
// The interface's own initial, which is what the packs spell — `l Log`,
// `s Store`, `p Pool`. Short because it appears in every call of every
// body and names something the signature beside it already declares.
func receiverIdent(iface Iface) string {
	if iface.Token == "" {
		return "subject"
	}
	return iface.Token[:1]
}

// discardOf drops a call's results where the body only asks whether the
// call returned: one blank per result.
func discardOf(m Method) string {
	n := len(m.Returns)
	if n == 0 {
		return ""
	}
	return strings.Repeat("_, ", n-1) + "_ ="
}

// errBindOf binds the error a body inspects, and is empty where the
// error is the only result — which the packs return directly rather
// than binding to a local used once on the next line.
func errBindOf(m Method) string {
	values := len(m.ValueReturns())
	if values == 0 {
		return ""
	}
	return strings.Repeat("_, ", values) + "err :="
}

// withheldBodies names the variants this inventory derived and no
// template renders yet, sorted, each once.
//
// Emitted into the header rather than logged: a consumer reading a
// short check list deserves to know whether the run derived little or
// spelled little, and those are different problems with different
// owners.
func withheldBodies(inv projection.Inventory) []string {
	seen := map[projection.BodyKind]bool{}
	var out []string
	for _, c := range inv.Checks {
		if c.ID.Method == "" || c.Body == nil {
			continue
		}
		kind := c.Body.BodyKind()
		if rendered()[kind] || seen[kind] {
			continue
		}
		seen[kind] = true
		out = append(out, strings.TrimPrefix(string(kind), projection.BodyKindPrefix))
	}
	sort.Strings(out)
	return out
}

// drawsFixture reports whether any check reads the run's fixture, which
// is what decides the builder's own parameter: the rows are closures
// over it, so one that draws cannot reach a fixture the builder was
// never handed.
func drawsFixture(checks []*CheckEmit) bool {
	for _, c := range checks {
		if c.Draws {
			return true
		}
	}
	return false
}

// valueBindOf binds a call's results where the body judges the first of
// them: `got, err :=`, one blank per result between.
func valueBindOf(m Method) string {
	values := len(m.ValueReturns())
	if values == 0 {
		return ""
	}
	return "got" + strings.Repeat(", _", values-1) + ", err :="
}

// valueDiscardOf blanks the results after the first, for a body judging
// one value from a method that reports no error.
//
// Empty where the method answers nothing: no body judges a result that
// does not exist, and the count would otherwise go negative.
func valueDiscardOf(m Method) string {
	if len(m.Returns) < 2 {
		return ""
	}
	return strings.Repeat(", _", len(m.Returns)-1)
}

// sentinelRef resolves a declared sentinel to the reference a body
// names it through.
//
// The annotator hands the parameter back QUALIFIED — which is right for
// identity and wrong for a call site, as [Method.MixinParam] says — so
// the qualifier is split back off and handed to the backend, which is
// what registers the import. A bare name means the interface's own
// package, which is where the declaration wrote it.
func sentinelRef(iface Iface, declared string) *sdk.Expr {
	pkg, symbol := iface.Package, declared
	if i := strings.LastIndex(declared, "."); i >= 0 {
		pkg, symbol = declared[:i], declared[i+1:]
	}
	return sdk.NewExternal(pkg, symbol)
}

// errStmtOf binds the error inside an if-statement's init, where a body
// judges it in the condition rather than after it.
func errStmtOf(m Method) string {
	return strings.Repeat("_, ", len(m.ValueReturns())) + "err :="
}

// zeroTypeOf is the reference a declared zero of a NAMED type needs
// rendering, nil where the type is predeclared or compares against nil.
//
// Through the backend rather than spelled here, because a named type
// from another package is an import this file has to register, and only
// the backend can register one.
func zeroTypeOf(m Method) *sdk.Expr {
	src := firstValueSource(m)
	if src == nil || src.Name == "" || golang.IsPredeclared(src.Name) {
		return nil
	}
	if ZeroShapeOf(m) == ZeroNil {
		return nil
	}
	return sdk.NewExternal(src.Package, src.Name)
}

// zeroWordOf is a predeclared type's own word — `string`, `int` — which
// needs no import and so no reference.
func zeroWordOf(m Method) string {
	src := firstValueSource(m)
	if src == nil || src.Name == "" || !golang.IsPredeclared(src.Name) {
		return ""
	}
	return src.Name
}

// firstValueSource is the result a zero-on-error body judges.
func firstValueSource(m Method) *node.TypeRef {
	values := m.ValueReturns()
	if len(values) == 0 {
		return nil
	}
	return values[0].Source
}
