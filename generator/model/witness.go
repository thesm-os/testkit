// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package model

import (
	"strings"

	"go.thesmos.sh/eidos/lang/golang"
	"go.thesmos.sh/eidos/sdk"

	"go.thesmos.sh/testkit/generator/internal/subject"
	"go.thesmos.sh/testkit/generator/suite"
)

// bindingsOf derives one interface's bindings, false where it reported why it
// could not.
// witnessedHarness rewrites a generic interface's projection at its
// witnesses: every parameter and return naming a type parameter lands at the
// concrete type, and the subject reference arrives instantiated. The
// projection is the suite's emit value, shared with every plugin that reads
// it, so the rewrite clones — methods, signatures and the subject alike —
// and the non-generic path returns the harness untouched.
func witnessedHarness(
	harness *suite.Contract, iface *sdk.Interface, witnesses []sdk.Ref,
) (*suite.Contract, map[string]string) {
	by := golang.WitnessBindings(iface.TypeParams, witnesses)
	if by == nil {
		return harness, nil
	}
	clone := *harness
	clone.IfaceRef = sdk.External(iface.Package, iface.Name, witnesses...)
	methods := make([]subject.Method, len(harness.Methods))
	for i := range harness.Methods {
		m := harness.Methods[i]
		sig := *m.Sig
		params := make([]golang.Param, len(sig.Params))
		for j, p := range sig.Params {
			p.Type = golang.SubstituteTypeParams(p.Type, by)
			params[j] = p
		}
		returns := make([]golang.Return, len(sig.Returns))
		for j, r := range sig.Returns {
			r.Type = golang.SubstituteTypeParams(r.Type, by)
			returns[j] = r
		}
		sig.Params, sig.Returns = params, returns
		m.Sig = &sig
		methods[i] = m
	}
	clone.Methods = methods

	q := make(map[string]string, len(by))
	for name, ref := range by {
		q[name] = witnessSpelling(ref)
	}
	return &clone, q
}

// witnessSpelling is a witness in the annotator's stamp vocabulary: bare for
// a builtin, package-qualified for anything else — the same form
// [golang.RefForQualified] lifts back into a reference.
func witnessSpelling(r sdk.Ref) string {
	if ext, qualified := r.(*sdk.ExternalRef); qualified {
		return ext.Package + "." + ext.Name
	}
	if b, builtin := r.(*sdk.BuiltinRef); builtin {
		return b.Name
	}
	return ""
}

// modelWitnesses resolves the concrete types a generic interface's property
// runs at, or reports the interface unusable after diagnosing why.
//
// Required rather than derived: the stub's companion only has to compile, so
// a derived palette serves it, but the property's pools, reference and laws
// all assert THROUGH these types — a silent guess would change the claim.
// Nothing here checks a witness satisfies its constraint; a wrong one is a
// compile error naming the type in the generated file, which is the best
// available outcome for a fact the generator cannot know.
func modelWitnesses(ctx *sdk.GeneratorContext, iface *sdk.Interface) ([]sdk.Ref, bool) {
	if len(iface.TypeParams) == 0 {
		return nil, true
	}
	dir := iface.Directive(DirectiveName)
	raw, given := "", false
	if dir != nil {
		raw, given = dir.KV[WitnessKey]
	}
	if !given {
		ctx.Diag.Errorf(iface.Pos(),
			"%s: interface %q is generic, and the property, the reference and "+
				"the pools all land at concrete types; name them with %s= — one "+
				"per type parameter, in declaration order",
			Name, iface.Name, WitnessKey)
		return nil, false
	}
	names := strings.Split(raw, ",")
	if len(names) != len(iface.TypeParams) {
		ctx.Diag.Errorf(iface.Pos(),
			"%s: %s=%q on %s names %d types for %d type parameters; supply one per parameter",
			Name, WitnessKey, raw, iface.Name, len(names), len(iface.TypeParams))
		return nil, false
	}
	out := make([]sdk.Ref, 0, len(names))
	for _, n := range names {
		out = append(out, golang.RefFor(strings.TrimSpace(n), iface.Package))
	}
	return out, true
}
